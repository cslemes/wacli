package api

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/steipete/wacli/internal/app"
	"github.com/steipete/wacli/internal/store"
	"github.com/steipete/wacli/internal/wa"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type sendTextRequest struct {
	To      string `json:"to" binding:"required"`
	Message string `json:"message" binding:"required"`
}

func sendTextHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sendTextRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		toJID, err := wa.ParseUserOrJID(req.To)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient: " + err.Error()})
			return
		}

		msgID, err := app.WA().SendText(ctx, toJID, req.Message)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "send failed: " + err.Error()})
			return
		}

		now := time.Now().UTC()
		chat := toJID
		chatName := app.WA().ResolveChatName(ctx, chat, "")
		kind := chatKindFromJID(chat)
		_ = app.DB().UpsertChat(chat.String(), kind, chatName, now)
		_ = app.DB().UpsertMessage(store.UpsertMessageParams{
			ChatJID:    chat.String(),
			ChatName:   chatName,
			MsgID:      string(msgID),
			SenderJID:  "",
			SenderName: "me",
			Timestamp:  now,
			FromMe:     true,
			Text:       req.Message,
		})

		c.JSON(http.StatusOK, gin.H{
			"sent": true,
			"to":   chat.String(),
			"id":   msgID,
		})
	}
}

type sendFileRequest struct {
	To      string `form:"to" binding:"required"`
	Caption string `form:"caption"`
}

func sendFileHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sendFileRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			return
		}
		defer file.Close()

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		toJID, err := wa.ParseUserOrJID(req.To)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient: " + err.Error()})
			return
		}

		// Save file temporarily
		tmpDir := os.TempDir()
		tmpPath := filepath.Join(tmpDir, fmt.Sprintf("wacli-upload-%d-%s", time.Now().Unix(), header.Filename))

		out, err := os.Create(tmpPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}
		_, err = io.Copy(out, file)
		out.Close()
		if err != nil {
			os.Remove(tmpPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}
		defer os.Remove(tmpPath)

		// Use the sendFile function from CLI
		msgID, _, err := sendFile(ctx, app, toJID, tmpPath, header.Filename, req.Caption, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "send failed: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"sent":     true,
			"to":       toJID.String(),
			"id":       msgID,
			"filename": header.Filename,
		})
	}
}

func chatKindFromJID(jid interface{}) string {
	jidStr := fmt.Sprintf("%v", jid)
	if len(jidStr) > 0 && jidStr[len(jidStr)-1] == 'g' {
		return "group"
	}
	return "dm"
}

// sendFile sends a file message (adapted from cmd/wacli/send_file.go)
func sendFile(ctx context.Context, a *app.App, to types.JID, filePath, filename, caption, mimeOverride string) (string, map[string]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", nil, err
	}

	name := strings.TrimSpace(filename)
	if name == "" {
		name = filepath.Base(filePath)
	}
	mimeType := strings.TrimSpace(mimeOverride)
	if mimeType == "" {
		mimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(filePath)))
	}
	if mimeType == "" {
		sniff := data
		if len(sniff) > 512 {
			sniff = sniff[:512]
		}
		mimeType = http.DetectContentType(sniff)
	}

	mediaType := "document"
	uploadType, _ := wa.MediaTypeFromString("document")
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		mediaType = "image"
		uploadType, _ = wa.MediaTypeFromString("image")
	case strings.HasPrefix(mimeType, "video/"):
		mediaType = "video"
		uploadType, _ = wa.MediaTypeFromString("video")
	case strings.HasPrefix(mimeType, "audio/"):
		mediaType = "audio"
		uploadType, _ = wa.MediaTypeFromString("audio")
	}

	up, err := a.WA().Upload(ctx, data, uploadType)
	if err != nil {
		return "", nil, err
	}

	now := time.Now().UTC()
	msg := &waProto.Message{}

	switch mediaType {
	case "image":
		msg.ImageMessage = &waProto.ImageMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
			Mimetype:      proto.String(mimeType),
			Caption:       proto.String(caption),
		}
	case "video":
		msg.VideoMessage = &waProto.VideoMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
			Mimetype:      proto.String(mimeType),
			Caption:       proto.String(caption),
		}
	case "audio":
		msg.AudioMessage = &waProto.AudioMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
			Mimetype:      proto.String(mimeType),
			PTT:           proto.Bool(false),
		}
	default:
		msg.DocumentMessage = &waProto.DocumentMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
			Mimetype:      proto.String(mimeType),
			FileName:      proto.String(name),
			Caption:       proto.String(caption),
			Title:         proto.String(name),
		}
	}

	id, err := a.WA().SendProtoMessage(ctx, to, msg)
	if err != nil {
		return "", nil, err
	}

	chatName := a.WA().ResolveChatName(ctx, to, "")
	kind := chatKindFromJID(to)
	_ = a.DB().UpsertChat(to.String(), kind, chatName, now)
	_ = a.DB().UpsertMessage(store.UpsertMessageParams{
		ChatJID:    to.String(),
		ChatName:   chatName,
		MsgID:      id,
		SenderJID:  "",
		SenderName: "me",
		Timestamp:  now,
		FromMe:     true,
		Text:       caption,
	})

	return id, map[string]string{
		"name":      name,
		"mime_type": mimeType,
		"media":     mediaType,
	}, nil
}

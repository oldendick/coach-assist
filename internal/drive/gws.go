package drive

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/oldendick/coach-assist/internal/config"
)

type GWSClient struct {
	gwsPath string
}

// MessagePart represents a part of a Gmail message (MIME structure).
type MessagePart struct {
	Filename string `json:"filename"`
	Body     struct {
		AttachmentID string `json:"attachmentId"`
	} `json:"body"`
	Parts []json.RawMessage `json:"parts"`
}

// NewGWSClient instantiates the formal execution wrapper bridging into external shell dependencies.
// It resolve the path to the 'gws' binary based on the platform and configuration.
func NewGWSClient(cfg *config.AppConfig) *GWSClient {
	path := "gws" // Default fallback to system PATH

	// 1. Check for manual override in config
	if cfg != nil && cfg.GWSPath != "" {
		path = cfg.GWSPath
	} else {
		// 2. Look for bundled binary in bin/ directory
		ext := ""
		if runtime.GOOS == "windows" {
			ext = ".exe"
		}
		// Try specific: bin/gws-<os>-<arch>[.exe], then try generic: bin/gws[.exe]
		specific := filepath.Join("bin", fmt.Sprintf("gws-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext))
		generic := filepath.Join("bin", "gws"+ext)

		for _, p := range []string{specific, generic} {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	return &GWSClient{gwsPath: path}
}

// Probe checks if the 'gws' binary is functional and authenticated by calling 'drive about get'.
func (g *GWSClient) Probe() error {
	_, err := g.run("drive", "about", "get", "--params", `{"fields": "user"}`)
	return err
}

// Login performs an interactive OAuth2 login flow using 'gws auth login'.
func (g *GWSClient) Login() error {
	fmt.Println("\n[!] Authentication Required")
	fmt.Println("Launching Google Workspace login flow...")
	fmt.Println("Please follow the instructions in your browser.")
	fmt.Println("-------------------------------------------")

	cmd := exec.Command(g.gwsPath, "auth", "login", "--services", "drive,gmail")

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := os.Environ()
	// Inject build-time secrets if they were provided via ldflags
	if ClientID != "" {
		env = append(env, "GOOGLE_WORKSPACE_CLI_CLIENT_ID="+ClientID)
	}
	if ClientSecret != "" {
		env = append(env, "GOOGLE_WORKSPACE_CLI_CLIENT_SECRET="+ClientSecret)
	}
	env = append(env, "GOOGLE_WORKSPACE_CLI_KEYRING_BACKEND=file")
	cmd.Env = env

	err := cmd.Run()
	if err == nil {
		fmt.Println("-------------------------------------------")
		fmt.Println("Login successful!")
	}
	return err
}

// Compile check
var _ WorkspaceService = (*GWSClient)(nil)

var (
	// ClientID and ClientSecret are populated at build-time via -ldflags if distributed as a bundle.
	ClientID     string
	ClientSecret string
)

// run safely dispatches the binary retaining local environment constants and macOS token bypass logic.
func (g *GWSClient) run(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, g.gwsPath, args...)

	env := os.Environ()
	// SSL_CERT_FILE is typically only needed for certain macOS/Linux environments 
	// when using 'gws' with its internal roots. On Windows, we skip it.
	if runtime.GOOS != "windows" {
		// Common Linux certificate paths
		certPaths := []string{
			"/etc/ssl/cert.pem",                   // macOS/Universal
			"/etc/ssl/certs/ca-certificates.crt",   // Debian/Ubuntu/Gentoo/Arch
			"/etc/pki/tls/certs/ca-bundle.crt",     // Fedora/RHEL/CentOS
			"/etc/ssl/ca-bundle.pem",               // OpenSUSE
		}
		for _, p := range certPaths {
			if _, err := os.Stat(p); err == nil {
				env = append(env, "SSL_CERT_FILE="+p)
				break
			}
		}
	}
	env = append(env, "GOOGLE_WORKSPACE_CLI_KEYRING_BACKEND=file")

	// Inject build-time secrets if they were provided via ldflags
	if ClientID != "" {
		env = append(env, "GOOGLE_WORKSPACE_CLI_CLIENT_ID="+ClientID)
	}
	if ClientSecret != "" {
		env = append(env, "GOOGLE_WORKSPACE_CLI_CLIENT_SECRET="+ClientSecret)
	}

	cmd.Env = env

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("gws command timed out after 30s")
		}
		return nil, fmt.Errorf("gws command failed: %v\nStderr: %s", err, stderr.String())
	}
	return out.Bytes(), nil
}

// DownloadLatestAttachment wraps the list -> get -> get-attachments pipeline.
// Returns the original attachment filename from the email.
func (g *GWSClient) DownloadLatestAttachment(subjectQuery, destFilename string, logFn func(string)) (string, error) {
	logFn(fmt.Sprintf("Scanning Gmail for Subject: '%s'...", subjectQuery))

	qBytes, _ := json.Marshal(map[string]string{
		"userId": "me",
		"q":      fmt.Sprintf(`subject:"%s" has:attachment`, subjectQuery),
	})

	listOut, err := g.run("gmail", "users", "messages", "list", "--params", string(qBytes))
	if err != nil {
		return "", fmt.Errorf("failed fetching message list: %w", err)
	}

	var rootList struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(listOut, &rootList); err != nil {
		return "", err
	}
	if len(rootList.Messages) == 0 {
		return "", fmt.Errorf("no emails matching '%s' found", subjectQuery)
	}

	msgID := rootList.Messages[0].ID
	logFn(fmt.Sprintf("Detected reliable thread reference: '%s'. Executing Google Drive MIME evaluation tree...", msgID))

	mBytes, _ := json.Marshal(map[string]string{"userId": "me", "id": msgID})
	getOut, err := g.run("gmail", "users", "messages", "get", "--params", string(mBytes))
	if err != nil {
		return "", err
	}

	var msgData struct {
		Payload MessagePart `json:"payload"`
	}
	_ = json.Unmarshal(getOut, &msgData)

	var attachID string
	var origFilename string
	var walk func(part MessagePart) (string, string)
	walk = func(part MessagePart) (string, string) {
		if part.Filename != "" && part.Body.AttachmentID != "" {
			return part.Body.AttachmentID, part.Filename
		}
		for _, raw := range part.Parts {
			var child MessagePart
			if json.Unmarshal(raw, &child) == nil {
				if id, fn := walk(child); id != "" {
					return id, fn
				}
			}
		}
		return "", ""
	}

	attachID, origFilename = walk(msgData.Payload)
	if attachID == "" {
		return "", fmt.Errorf("failed to locate attachment in MIME tree")
	}

	logFn(fmt.Sprintf("Found attachment: '%s' (ID: %s). Downloading...", origFilename, attachID))

	aBytes, _ := json.Marshal(map[string]string{
		"userId":    "me",
		"messageId": msgID,
		"id":        attachID,
	})
	blobOut, err := g.run("gmail", "users", "messages", "attachments", "get", "--params", string(aBytes))
	if err != nil {
		return "", err
	}

	var blobData struct {
		Data string `json:"data"`
	}
	_ = json.Unmarshal(blobOut, &blobData)

	logFn(fmt.Sprintf("Received massive binary block (%d bytes string length) safely via REST buffer.", len(blobData.Data)))

	// Handle GWS unpadded base64 edge-cases
	encoded := blobData.Data
	for len(encoded)%4 != 0 {
		encoded += "="
	}

	fileBytes, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64url payload parsing failure: %w", err)
	}

	// Ensure artifacts/ directory exists
	if err := os.MkdirAll("artifacts", 0755); err != nil {
		return "", fmt.Errorf("failed creating artifacts dir: %w", err)
	}

	// Write the original file with its real name
	origPath := filepath.Join("artifacts", origFilename)
	if err := os.WriteFile(origPath, fileBytes, 0644); err != nil {
		return "", fmt.Errorf("failed writing original file: %w", err)
	}
	logFn(fmt.Sprintf("Saved original: artifacts/%s", origFilename))

	// Copy to canonical latest-* name
	canonicalPath := filepath.Join("artifacts", destFilename)
	if err := os.WriteFile(canonicalPath, fileBytes, 0644); err != nil {
		return "", fmt.Errorf("failed writing canonical copy: %w", err)
	}
	logFn(fmt.Sprintf("Copied to canonical: artifacts/%s", destFilename))

	return origFilename, nil
}

// ListFolderContents returns all non-trashed children in a Drive folder.
func (g *GWSClient) ListFolderContents(parentFolderID string) ([]DriveItem, error) {
	q := fmt.Sprintf("'%s' in parents and trashed = false", parentFolderID)
	params, _ := json.Marshal(map[string]interface{}{
		"q":                         q,
		"supportsAllDrives":         true,
		"includeItemsFromAllDrives": true,
		"pageSize":                  1000,
	})

	out, err := g.run("drive", "files", "list", "--params", string(params))
	if err != nil {
		return nil, fmt.Errorf("drive files list failed: %w", err)
	}

	var resp struct {
		Files []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"files"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing drive response: %w", err)
	}

	items := make([]DriveItem, 0, len(resp.Files))
	for _, f := range resp.Files {
		items = append(items, DriveItem{Name: f.Name, ID: f.ID})
	}
	return items, nil
}

// ExportFile uses 'gws' to convert a Google Workspace file and save it locally.
func (g *GWSClient) ExportFile(id, mimeType, destPath string) error {
	params, _ := json.Marshal(map[string]string{
		"fileId":   id,
		"mimeType": mimeType,
	})
	_, err := g.run("drive", "files", "export", "--params", string(params), "--output", destPath)
	return err
}

// DownloadFile uses 'gws' to download binary content directly.
func (g *GWSClient) DownloadFile(id, destPath string) error {
	params, _ := json.Marshal(map[string]string{
		"fileId": id,
		"alt":    "media",
	})
	_, err := g.run("drive", "files", "get", "--params", string(params), "--output", destPath)
	return err
}

// SearchFiles wraps 'drive files list' with a custom query and returns all non-trashed matches.
func (g *GWSClient) SearchFiles(query string) ([]DriveItem, error) {
	params, _ := json.Marshal(map[string]interface{}{
		"q":                         query,
		"supportsAllDrives":         true,
		"includeItemsFromAllDrives": true,
		"pageSize":                  100,
	})

	out, err := g.run("drive", "files", "list", "--params", string(params))
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var resp struct {
		Files []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"files"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}

	items := make([]DriveItem, 0, len(resp.Files))
	for _, f := range resp.Files {
		items = append(items, DriveItem{Name: f.Name, ID: f.ID})
	}
	return items, nil
}

// CreateFolder creates a new folder in Google Drive.
func (g *GWSClient) CreateFolder(parentID, name string) (string, error) {
	params, _ := json.Marshal(map[string]bool{"supportsAllDrives": true})
	body, _ := json.Marshal(map[string]interface{}{
		"name":     name,
		"mimeType": "application/vnd.google-apps.folder",
		"parents":  []string{parentID},
	})

	out, err := g.run("drive", "files", "create", "--params", string(params), "--json", string(body))
	if err != nil {
		return "", fmt.Errorf("failed creating folder: %w", err)
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", err
	}
	return resp.ID, nil
}

// CopyFile creates a copy of an existing file in the specified parent folder with a new name.
func (g *GWSClient) CopyFile(fileID, parentID, newName string) (string, error) {
	params, _ := json.Marshal(map[string]interface{}{
		"fileId":            fileID,
		"supportsAllDrives": true,
	})
	body, _ := json.Marshal(map[string]interface{}{
		"name":    newName,
		"parents": []string{parentID},
	})

	out, err := g.run("drive", "files", "copy", "--params", string(params), "--json", string(body))
	if err != nil {
		return "", fmt.Errorf("failed copying file: %w", err)
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", err
	}
	return resp.ID, nil
}

// CreatePermission adds a new permission to a file or folder.
func (g *GWSClient) CreatePermission(fileID, role, pType string) error {
	params, _ := json.Marshal(map[string]interface{}{
		"fileId":            fileID,
		"supportsAllDrives": true,
	})
	body, _ := json.Marshal(map[string]string{
		"role": role,
		"type": pType,
	})

	_, err := g.run("drive", "permissions", "create", "--params", string(params), "--json", string(body))
	return err
}

// UpdateSheetValues updates multiple ranges in a Google Sheet in a single batch.
func (g *GWSClient) UpdateSheetValues(spreadsheetID string, updates []SheetUpdate) error {
	params, _ := json.Marshal(map[string]string{
		"spreadsheetId": spreadsheetID,
	})

	type entry struct {
		Range  string          `json:"range"`
		Values [][]interface{} `json:"values"`
	}
	data := make([]entry, len(updates))
	for i, u := range updates {
		data[i] = entry{Range: u.Range, Values: u.Values}
	}

	body, _ := json.Marshal(map[string]interface{}{
		"valueInputOption": "USER_ENTERED",
		"data":             data,
	})

	_, err := g.run("sheets", "spreadsheets", "values", "batchUpdate", "--params", string(params), "--json", string(body))
	return err
}

// GetSheetValues retrieves values from a specific range in a Google Sheet.
func (g *GWSClient) GetSheetValues(spreadsheetID, rangeStr string) ([][]interface{}, error) {
	params, _ := json.Marshal(map[string]string{
		"spreadsheetId": spreadsheetID,
		"range":         rangeStr,
	})

	out, err := g.run("sheets", "spreadsheets", "values", "get", "--params", string(params))
	if err != nil {
		return nil, fmt.Errorf("failed getting sheet values: %w", err)
	}

	var resp struct {
		Values [][]interface{} `json:"values"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}
	return resp.Values, nil
}

// GetSpreadsheetMetadata retrieves structural metadata (like merges) for a spreadsheet.
func (g *GWSClient) GetSpreadsheetMetadata(spreadsheetID string) (*SpreadsheetMetadata, error) {
	params, _ := json.Marshal(map[string]string{
		"spreadsheetId": spreadsheetID,
		"fields":        "sheets(properties.title,merges)",
	})

	out, err := g.run("sheets", "spreadsheets", "get", "--params", string(params))
	if err != nil {
		return nil, fmt.Errorf("failed getting spreadsheet metadata: %w", err)
	}

	var meta SpreadsheetMetadata
	if err := json.Unmarshal(out, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// SearchMessages searches Gmail for messages matching a query.
func (g *GWSClient) SearchMessages(query string) ([]MessageSummary, error) {
	qBytes, _ := json.Marshal(map[string]string{
		"userId": "me",
		"q":      query,
	})

	listOut, err := g.run("gmail", "users", "messages", "list", "--params", string(qBytes))
	if err != nil {
		return nil, fmt.Errorf("failed fetching message list: %w", err)
	}

	var rootList struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(listOut, &rootList); err != nil {
		return nil, err
	}

	results := make([]MessageSummary, 0, len(rootList.Messages))
	// Limit to top 10 for performance
	limit := len(rootList.Messages)
	if limit > 10 {
		limit = 10
	}

	for i := 0; i < limit; i++ {
		msgID := rootList.Messages[i].ID
		mBytes, _ := json.Marshal(map[string]interface{}{
			"userId":          "me",
			"id":              msgID,
			"format":          "metadata",
			"metadataHeaders": []string{"Subject", "Date"},
		})
		getOut, err := g.run("gmail", "users", "messages", "get", "--params", string(mBytes))
		if err != nil {
			continue
		}

		var msgData struct {
			ID      string `json:"id"`
			Snippet string `json:"snippet"`
			Payload struct {
				Headers []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"headers"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(getOut, &msgData); err != nil {
			continue
		}

		summary := MessageSummary{ID: msgID, Snippet: msgData.Snippet}
		for _, h := range msgData.Payload.Headers {
			if h.Name == "Subject" {
				summary.Subject = h.Value
			} else if h.Name == "Date" {
				summary.Date = h.Value
			}
		}
		results = append(results, summary)
	}

	return results, nil
}

// GetMessageAttachments returns metadata for all attachments in a message.
func (g *GWSClient) GetMessageAttachments(messageID string) ([]AttachmentInfo, error) {
	mBytes, _ := json.Marshal(map[string]string{"userId": "me", "id": messageID})
	getOut, err := g.run("gmail", "users", "messages", "get", "--params", string(mBytes))
	if err != nil {
		return nil, err
	}

	var msgData struct {
		Payload MessagePart `json:"payload"`
	}
	if err := json.Unmarshal(getOut, &msgData); err != nil {
		return nil, err
	}

	var attachments []AttachmentInfo
	var walk func(part MessagePart)
	walk = func(part MessagePart) {
		if part.Filename != "" && part.Body.AttachmentID != "" {
			attachments = append(attachments, AttachmentInfo{
				ID:       part.Body.AttachmentID,
				Filename: part.Filename,
			})
		}
		for _, raw := range part.Parts {
			var child MessagePart
			if json.Unmarshal(raw, &child) == nil {
				walk(child)
			}
		}
	}
	walk(msgData.Payload)

	return attachments, nil
}

// DownloadAttachment downloads a specific attachment from a message.
func (g *GWSClient) DownloadAttachment(messageID, attachmentID, destFilename string) error {
	aBytes, _ := json.Marshal(map[string]string{
		"userId":    "me",
		"messageId": messageID,
		"id":        attachmentID,
	})
	blobOut, err := g.run("gmail", "users", "messages", "attachments", "get", "--params", string(aBytes))
	if err != nil {
		return err
	}

	var blobData struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(blobOut, &blobData); err != nil {
		return err
	}

	encoded := blobData.Data
	for len(encoded)%4 != 0 {
		encoded += "="
	}

	fileBytes, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("base64url payload parsing failure: %w", err)
	}

	if err := os.MkdirAll("artifacts", 0755); err != nil {
		return fmt.Errorf("failed creating artifacts dir: %w", err)
	}

	canonicalPath := filepath.Join("artifacts", destFilename)
	if err := os.WriteFile(canonicalPath, fileBytes, 0644); err != nil {
		return fmt.Errorf("failed writing canonical copy: %w", err)
	}

	return nil
}

// CreateDraft creates a new draft email in Gmail.
func (g *GWSClient) CreateDraft(from, to, cc, subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nCc: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s", from, to, cc, subject, body)

	draftsDir := filepath.Join("artifacts", "drafts")
	if err := os.MkdirAll(draftsDir, 0755); err != nil {
		return fmt.Errorf("failed to create drafts directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("draft_%s.eml", timestamp)
	filePath := filepath.Join(draftsDir, filename)

	if err := os.WriteFile(filePath, []byte(msg), 0644); err != nil {
		return fmt.Errorf("failed to write .eml file: %w", err)
	}

	params, _ := json.Marshal(map[string]string{"userId": "me"})
	_, err := g.run("gmail", "users", "drafts", "create",
		"--params", string(params),
		"--upload", filePath,
		"--upload-content-type", "message/rfc822")
	return err
}

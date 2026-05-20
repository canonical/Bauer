package figma_test

import (
	"testing"

	"bauer/internal/figma"
)

func TestParseLink(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		fileKey string
		nodeID  string
		wantErr bool
	}{
		{
			name:    "file URL without node",
			input:   "https://www.figma.com/file/bwqWjuxIJiwDetRL2fYwNN/Product-Name",
			fileKey: "bwqWjuxIJiwDetRL2fYwNN",
			nodeID:  "",
		},
		{
			name:    "file URL with node",
			input:   "https://www.figma.com/file/bwqWjuxIJiwDetRL2fYwNN/Product-Name?node-id=1%3A42",
			fileKey: "bwqWjuxIJiwDetRL2fYwNN",
			nodeID:  "1:42",
		},
		{
			name:    "design URL with node",
			input:   "https://www.figma.com/design/bwqWjuxIJiwDetRL2fYwNN/Product?node-id=6039-4970",
			fileKey: "bwqWjuxIJiwDetRL2fYwNN",
			nodeID:  "6039-4970",
		},
		{
			name:    "not a figma URL",
			input:   "https://www.example.com/something",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := figma.ParseLink(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseLink(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if ref.FileKey != tt.fileKey {
				t.Errorf("FileKey = %q, want %q", ref.FileKey, tt.fileKey)
			}
			if ref.NodeID != tt.nodeID {
				t.Errorf("NodeID = %q, want %q", ref.NodeID, tt.nodeID)
			}
		})
	}
}

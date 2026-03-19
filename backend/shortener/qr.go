package shortener

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

// GenerateQR creates a QR code PNG encoding the given URL at the requested size.
// Pass the full short URL (e.g. "https://example.com/abc123") so the QR code
// resolves correctly when scanned. If url is just a short code the QR will only
// display the code as plain text.
func GenerateQR(url string, size int) ([]byte, error) {
	qrCode, err := qr.Encode(url, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("qr encode: %w", err)
	}

	qrCode, err = barcode.Scale(qrCode, size, size)
	if err != nil {
		return nil, fmt.Errorf("qr scale: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, qrCode); err != nil {
		return nil, fmt.Errorf("png encode: %w", err)
	}

	return buf.Bytes(), nil
}

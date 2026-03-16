package shortener

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

// GenerateQR creates a QR code PNG for the given short code.
// The code is encoded as a relative path (e.g., "/abc123") so the
// consuming application can prepend the appropriate base URL.
func GenerateQR(code string, size int) ([]byte, error) {
	qrCode, err := qr.Encode(code, qr.M, qr.Auto)
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

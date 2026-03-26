package tray

// generateICO creates a minimal 16x16 ICO file with a solid colored circle
// on a transparent background. This avoids the need for external .ico files.
func generateICO(r, g, b byte) []byte {
	const (
		width  = 16
		height = 16
	)

	// BMP pixel data: 16x16 BGRA, bottom-up row order.
	pixels := make([]byte, width*height*4)
	centerX, centerY := float64(width)/2, float64(height)/2
	radius := float64(width)/2 - 1

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Bottom-up row order for BMP.
			row := height - 1 - y
			offset := (row*width + x) * 4

			// Distance from center.
			dx := float64(x) - centerX + 0.5
			dy := float64(y) - centerY + 0.5
			dist := dx*dx + dy*dy

			if dist <= radius*radius {
				// Inside circle: BGRA order.
				pixels[offset+0] = b // blue
				pixels[offset+1] = g // green
				pixels[offset+2] = r // red
				pixels[offset+3] = 255 // alpha
			} else {
				// Outside circle: transparent.
				pixels[offset+0] = 0
				pixels[offset+1] = 0
				pixels[offset+2] = 0
				pixels[offset+3] = 0
			}
		}
	}

	// AND mask (1-bit transparency mask, all zeros = fully opaque, handled by alpha).
	andMask := make([]byte, height*((width+31)/32)*4)

	// BMP info header (BITMAPINFOHEADER, 40 bytes).
	bmpHeader := []byte{
		40, 0, 0, 0, // biSize
		byte(width), 0, 0, 0, // biWidth
		byte(height * 2), 0, 0, 0, // biHeight (doubled for ICO: XOR + AND)
		1, 0, // biPlanes
		32, 0, // biBitCount (32-bit BGRA)
		0, 0, 0, 0, // biCompression (BI_RGB)
		0, 0, 0, 0, // biSizeImage (can be 0 for BI_RGB)
		0, 0, 0, 0, // biXPelsPerMeter
		0, 0, 0, 0, // biYPelsPerMeter
		0, 0, 0, 0, // biClrUsed
		0, 0, 0, 0, // biClrImportant
	}

	imageData := append(bmpHeader, pixels...)
	imageData = append(imageData, andMask...)

	imageSize := len(imageData)
	dataOffset := 6 + 16 // ICO header (6) + one directory entry (16)

	// ICO file header (6 bytes).
	icoHeader := []byte{
		0, 0, // reserved
		1, 0, // type: 1 = ICO
		1, 0, // count: 1 image
	}

	// ICO directory entry (16 bytes).
	icoEntry := []byte{
		byte(width),  // width (0 = 256)
		byte(height), // height (0 = 256)
		0,            // color palette count
		0,            // reserved
		1, 0,         // color planes
		32, 0, // bits per pixel
		byte(imageSize), byte(imageSize >> 8), byte(imageSize >> 16), byte(imageSize >> 24), // image size
		byte(dataOffset), byte(dataOffset >> 8), byte(dataOffset >> 16), byte(dataOffset >> 24), // data offset
	}

	ico := append(icoHeader, icoEntry...)
	ico = append(ico, imageData...)

	return ico
}

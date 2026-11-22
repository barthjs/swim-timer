// Package colormix mixes colors the way the CSS function color-mix does,
// so the server derives the same shades as the browser.
package colormix

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ErrInvalidHex reports a color that is no hex color.
var ErrInvalidHex = errors.New("color is no hex color")

// WithBlack mixes a hex color with black as color-mix(in oklch, black <percent>%, <hex>) does
// and returns the result as an opaque hex color of six digits.
func WithBlack(hex string, percent float64) (string, error) {
	red, green, blue, err := parseHex(hex)
	if err != nil {
		return "", err
	}

	// Black has no hue, so the mix keeps the hue of the color and scales its lightness and chroma,
	// which in OKLab scales every component.
	lightness, a, b := linearToOKLab(toLinear(red), toLinear(green), toLinear(blue))
	weight := 1 - percent/100
	linearRed, linearGreen, linearBlue := okLabToLinear(lightness*weight, a*weight, b*weight)

	return formatHex(fromLinear(linearRed), fromLinear(linearGreen), fromLinear(linearBlue)), nil
}

// parseHex returns the color channels of a hex color of three, four, six, or eight digits between 0 and 1.
func parseHex(hex string) (float64, float64, float64, error) {
	digits, ok := strings.CutPrefix(hex, "#")
	if !ok {
		return 0, 0, 0, fmt.Errorf("%w: %q", ErrInvalidHex, hex)
	}

	switch len(digits) {
	case 3, 4:
		digits = string([]byte{digits[0], digits[0], digits[1], digits[1], digits[2], digits[2]})
	case 6, 8:
		digits = digits[:6]
	default:
		return 0, 0, 0, fmt.Errorf("%w: %q", ErrInvalidHex, hex)
	}

	value, err := strconv.ParseUint(digits, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: %q", ErrInvalidHex, hex)
	}

	return float64(value>>16&0xff) / 255, float64(value>>8&0xff) / 255, float64(value&0xff) / 255, nil
}

// toLinear removes the gamma of an sRGB channel.
func toLinear(channel float64) float64 {
	if channel <= 0.04045 {
		return channel / 12.92
	}
	return math.Pow((channel+0.055)/1.055, 2.4)
}

// linearToOKLab converts linear sRGB to OKLab with the matrices of its definition.
func linearToOKLab(red, green, blue float64) (float64, float64, float64) {
	l := math.Cbrt(0.4122214708*red + 0.5363325363*green + 0.0514459929*blue)
	m := math.Cbrt(0.2119034982*red + 0.6806995451*green + 0.1073969566*blue)
	s := math.Cbrt(0.0883024619*red + 0.2817188376*green + 0.6299787005*blue)

	return 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s
}

// okLabToLinear converts OKLab back to linear sRGB.
func okLabToLinear(lightness, a, b float64) (float64, float64, float64) {
	l := cube(lightness + 0.3963377774*a + 0.2158037573*b)
	m := cube(lightness - 0.1055613458*a - 0.0638541728*b)
	s := cube(lightness - 0.0894841775*a - 1.2914855480*b)

	return 4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s
}

// cube returns the third power of a value.
func cube(value float64) float64 {
	return value * value * value
}

// fromLinear applies the gamma of sRGB to a linear channel and keeps it within the gamut.
func fromLinear(channel float64) float64 {
	channel = min(max(channel, 0), 1)
	if channel <= 0.0031308 {
		return channel * 12.92
	}
	return 1.055*math.Pow(channel, 1/2.4) - 0.055
}

// formatHex returns channels between 0 and 1 as a hex color.
func formatHex(red, green, blue float64) string {
	return fmt.Sprintf("#%02x%02x%02x", toByte(red), toByte(green), toByte(blue))
}

// toByte rounds a channel between 0 and 1 to a byte.
func toByte(channel float64) uint8 {
	return uint8(math.Round(channel * 255))
}

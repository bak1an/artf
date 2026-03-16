package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

const (
	defaultMaxUploadSize = "100m"
	maxUploadSizeKey     = "max-upload-size"
)

func resolveMaxUploadSize(v *viper.Viper) (int64, error) {
	return parseByteSize(v.GetString(maxUploadSizeKey))
}

func parseByteSize(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("size must not be empty")
	}

	multiplier := int64(1)
	last := value[len(value)-1]

	if last < '0' || last > '9' {
		switch strings.ToLower(string(last)) {
		case "k":
			multiplier = 1024
		case "m":
			multiplier = 1024 * 1024
		case "g":
			multiplier = 1024 * 1024 * 1024
		default:
			return 0, fmt.Errorf("unsupported size suffix %q", string(last))
		}
		value = value[:len(value)-1]
		if value == "" {
			return 0, fmt.Errorf("size is missing a numeric value")
		}
	}

	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q", raw)
	}
	if n <= 0 {
		return 0, fmt.Errorf("size must be greater than zero")
	}
	if n > (1<<63-1)/multiplier {
		return 0, fmt.Errorf("size %q overflows int64", raw)
	}

	return n * multiplier, nil
}

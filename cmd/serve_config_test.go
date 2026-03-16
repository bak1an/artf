package cmd

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{name: "bytes", input: "100", want: 100},
		{name: "kilobytes", input: "1k", want: 1024},
		{name: "megabytes", input: "100m", want: 100 * 1024 * 1024},
		{name: "gigabytes uppercase", input: "2G", want: 2 * 1024 * 1024 * 1024},
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "unsupported suffix", input: "10t", wantErr: true},
		{name: "invalid", input: "abc", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseByteSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseByteSize(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("parseByteSize(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveMaxUploadSize(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		v := viper.New()
		v.SetDefault(maxUploadSizeKey, defaultMaxUploadSize)

		got, err := resolveMaxUploadSize(v)
		if err != nil {
			t.Fatalf("resolveMaxUploadSize() error = %v", err)
		}
		if got != 100*1024*1024 {
			t.Fatalf("resolveMaxUploadSize() = %d, want %d", got, 100*1024*1024)
		}
	})

	t.Run("env override", func(t *testing.T) {
		v := viper.New()
		v.SetDefault(maxUploadSizeKey, defaultMaxUploadSize)
		if err := v.BindEnv(maxUploadSizeKey, "ARTF_MAX_UPLOAD_SIZE"); err != nil {
			t.Fatalf("BindEnv: %v", err)
		}

		t.Setenv("ARTF_MAX_UPLOAD_SIZE", "1g")

		got, err := resolveMaxUploadSize(v)
		if err != nil {
			t.Fatalf("resolveMaxUploadSize() error = %v", err)
		}
		if got != 1024*1024*1024 {
			t.Fatalf("resolveMaxUploadSize() = %d, want %d", got, 1024*1024*1024)
		}
	})

	t.Run("flag binding", func(t *testing.T) {
		fs := pflag.NewFlagSet("serve", pflag.ContinueOnError)
		fs.String(maxUploadSizeKey, defaultMaxUploadSize, "")

		v := viper.New()
		if err := v.BindPFlags(fs); err != nil {
			t.Fatalf("BindPFlags: %v", err)
		}

		if err := fs.Set(maxUploadSizeKey, "256m"); err != nil {
			t.Fatalf("Set flag: %v", err)
		}

		got, err := resolveMaxUploadSize(v)
		if err != nil {
			t.Fatalf("resolveMaxUploadSize() error = %v", err)
		}
		if got != 256*1024*1024 {
			t.Fatalf("resolveMaxUploadSize() = %d, want %d", got, 256*1024*1024)
		}
	})
}

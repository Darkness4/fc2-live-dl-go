//go:build unit

package utils_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/utils"
	"github.com/stretchr/testify/require"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		title    string
	}{
		{
			input:    "/home/user/プロジェクト/2023年/3月/プロジェクト2023年3月16日_15時30分59秒-これは本当に長いファイル名です_!@#$%^&()_禁止文字と予約済み名前(LPT1, PRN, .config)を含みます.txt",
			expected: "_home_user_プロジェクト_2023年_3月_プロジェクト2023年3月16日_15時30分59秒-これは本当に長いファイル名です_!@#$%^&()_禁止文字と予約済み名前(LPT1, PRN, .config)を含みます.txt",
			title:    "Positive test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual := utils.SanitizeFilename(tt.input)

			// Assert
			require.Equal(t, tt.expected, actual)
		})
	}
}

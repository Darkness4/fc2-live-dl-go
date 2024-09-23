package fc2

import (
	"bytes"
	"text/template"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/Darkness4/fc2-live-dl-go/utils"
)

// FormatOutput formats the output file name.
func FormatOutput(
	outFormat string,
	meta *api.GetMetaData,
	labels map[string]string,
	ext string,
) (string, error) {
	timeNow := time.Now()
	formatInfo := struct {
		ChannelID   string
		ChannelName string
		Date        string
		Time        string
		Title       string
		Ext         string
		MetaData    *api.GetMetaData
		Labels      map[string]string
	}{
		Date:   timeNow.Format("2006-01-02"),
		Time:   timeNow.Format("150405"),
		Ext:    ext,
		Labels: labels,
	}

	tmpl, err := template.New("gotpl").Parse(outFormat)
	if err != nil {
		return "", err
	}

	if meta != nil {
		formatInfo.ChannelID = utils.SanitizeFilename(meta.ChannelData.ChannelID)
		formatInfo.ChannelName = utils.SanitizeFilename(meta.ProfileData.Name)
		formatInfo.Title = utils.SanitizeFilename(meta.ChannelData.Title)
		formatInfo.MetaData = meta
	}

	var formatted bytes.Buffer
	if err = tmpl.Execute(&formatted, formatInfo); err != nil {
		return "", err
	}

	return formatted.String(), nil
}

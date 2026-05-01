package recorder

import (
	"fmt"
	"time"

	gttelemetry "github.com/zetetos/gt-telemetry/v2"
)

type Recorder struct {
	client  *gttelemetry.Client
	dataDir string
}

func New(client *gttelemetry.Client, dataDir string) *Recorder {
	return &Recorder{
		client:  client,
		dataDir: dataDir,
	}
}

func (r *Recorder) Start(filename string) error {
	path := r.dataDir + "/" + filename
	return r.client.StartRecording(path)
}

func (r *Recorder) StartTimestamped(prefix, ext string) (string, error) {
	name := fmt.Sprintf("%s_%s.%s", prefix, time.Now().Format("20060102_150405"), ext)
	return name, r.Start(name)
}

func (r *Recorder) Stop() error {
	return r.client.StopRecording()
}

func (r *Recorder) IsRecording() bool {
	return r.client.IsRecording()
}

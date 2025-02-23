package cookie

import (
	"fmt"
	"os"

	"github.com/shamaton/msgpack/v2"
)

// load loads the jar from a file.
func (j *Jar) load() error {
	if _, err := os.Stat(j.filename); os.IsNotExist(err) {
		return nil
	}

	f, err := os.Open(j.filename)
	if err != nil {
		return fmt.Errorf("cannot open %s: %v", j.filename, err)
	}
	defer f.Close()

	// Deserialize
	j.mu.Lock()
	defer j.mu.Unlock()
	return msgpack.UnmarshalRead(f, &j.entries)
}

// Save saves the jar to a file.
func (j *Jar) Save() error {
	f, err := os.OpenFile(j.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("cannot open %s: %v", j.filename, err)
	}
	defer f.Close()

	// Serialize
	j.mu.Lock()
	defer j.mu.Unlock()
	d, err := msgpack.Marshal(j.entries)
	if err != nil {
		return err
	}
	_, err = f.Write(d)
	return err
}

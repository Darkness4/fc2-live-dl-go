package cookie

import (
	"fmt"
	"os"

	"github.com/Darkness4/fc2-live-dl-go/crypto"
	"github.com/shamaton/msgpack/v3"
)

// load loads the jar from a file.
func (j *Jar) load() error {
	if _, err := os.Stat(j.filename); os.IsNotExist(err) {
		return nil
	}

	f, err := os.Open(j.filename)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", j.filename, err)
	}
	defer f.Close()

	data, err := crypto.Decrypt(f, []byte(j.encryptionSecret))
	if err != nil {
		return err
	}

	// Deserialize
	j.mu.Lock()
	defer j.mu.Unlock()

	return msgpack.Unmarshal(data, &j.entries)
}

// Exists returns true if the jar exists.
func (j *Jar) Exists() bool {
	_, err := os.Stat(j.filename)
	return err == nil
}

// Save saves the jar to a file.
func (j *Jar) Save() error {
	f, err := os.OpenFile(j.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", j.filename, err)
	}
	defer f.Close()

	// Serialize
	j.mu.Lock()
	defer j.mu.Unlock()
	d, err := msgpack.Marshal(j.entries)
	if err != nil {
		return err
	}

	return crypto.Encrypt(f, []byte(j.encryptionSecret), d)
}

// Delete deletes the jar file.
func (j *Jar) Delete() {
	_ = os.Remove(j.filename)
}

package watch_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/cmd/watch"
	"github.com/stretchr/testify/require"
)

func TestConfigReloaderTwoConfigs(t *testing.T) {
	// Create a new parent context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a temporary directory to store the config file
	tempDir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a temporary config file and write some data to it
	configFile := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configFile, []byte(`channels:
  '40740626':
    labels:
      EnglishName: Komae Nadeshiko
`), 0644)
	require.NoError(t, err)

	// Create a config channel and start observing the config file
	configChan := make(chan *watch.Config)
	go watch.ObserveConfig(ctx, configFile, configChan)

	// Create a mock handleConfig function that just sleeps for 1 second
	handleConfigCallCount := 0
	handleConfigCalls := make([]*watch.Config, 2)
	doneChan := make(chan struct{})
	handleConfigMock := func(ctx context.Context, cfg *watch.Config) {
		handleConfigCalls[handleConfigCallCount] = cfg
		handleConfigCallCount++
		select {
		case doneChan <- struct{}{}:
			return
		case <-ctx.Done():
			return
		}
	}

	// Launch the configReloader function in a separate goroutine
	wg := sync.WaitGroup{}
	wg.Go(func() {
		err := watch.ConfigReloader(ctx, configChan, handleConfigMock)
		require.Equal(t, context.Canceled, err)
	})

	// Wait for the handleConfig call to complete
	<-doneChan

	// Write a new config file with different data
	time.Sleep(time.Second)
	err = os.WriteFile(configFile, []byte(`channels:
  '40740626':
    labels:
      EnglishName: Komae Nadeshiko
  '72364867':
    labels:
      EnglishName: Uno Sakura
`), 0644)
	require.NoError(t, err)

	// Wait for the second handleConfig call to complete
	<-doneChan

	// Check that handleConfig was called twice with the correct configs
	require.Equal(t, 2, handleConfigCallCount)
	require.Equal(t, 1, len(handleConfigCalls[0].Channels))
	require.Equal(t, 2, len(handleConfigCalls[1].Channels))

	// Cancel the parent context to stop the configReloader function
	cancel()

	// Wait for the configReloader function to exit
	wg.Wait()
}

func TestConfigReloaderCancellation(t *testing.T) {
	// Create a new parent context
	ctx, cancel := context.WithCancel(context.Background())

	// Create a temporary directory to store the config file
	tempDir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a temporary config file and write some data to it
	configFile := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configFile, []byte(`channels:
  '40740626':
    labels:
      EnglishName: Komae Nadeshiko
`), 0644)
	require.NoError(t, err)

	// Create a config channel and start observing the config file
	configChan := make(chan *watch.Config)
	go watch.ObserveConfig(ctx, configFile, configChan)

	// Create a mock handleConfig function that just sleeps for 1 second
	handleConfigCallCount := 0
	handleConfigCalls := make([]*watch.Config, 2)
	readyChan := make(chan struct{})
	doneChan := make(chan struct{})
	handleConfigMock := func(ctx context.Context, cfg *watch.Config) {
		handleConfigCalls[handleConfigCallCount] = cfg
		handleConfigCallCount++
		fmt.Println("waiting")
		readyChan <- struct{}{}
		<-ctx.Done()
		fmt.Println("finishing")
		doneChan <- struct{}{}
	}

	// Launch the configReloader function in a separate goroutine
	wg := sync.WaitGroup{}
	wg.Go(func() {
		err := watch.ConfigReloader(ctx, configChan, handleConfigMock)
		require.Equal(t, context.Canceled, err)
	})

	<-readyChan

	// Cancel the parent context to stop the configReloader function
	cancel()

	// Wait for the second handleConfig call to complete
	<-doneChan

	// Check that handleConfig was called twice with the correct configs
	require.Equal(t, 1, handleConfigCallCount)
	require.Equal(t, 1, len(handleConfigCalls[0].Channels))

	// Wait for the configReloader function to exit
	wg.Wait()
}

package cfcomponent

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestReadingFromJsonFile(t *testing.T) {
	file, err := ioutil.TempFile("", "config")
	defer func() {
		os.Remove(file.Name())
	}()
	assert.NoError(t, err)
	_, err = file.Write([]byte(`{"VarzUser":"User"}`))
	assert.NoError(t, err)

	err = file.Close()
	assert.NoError(t, err)

	config := &Config{}
	err = ReadConfigInto(config, file.Name())
	assert.NoError(t, err)

	assert.Equal(t, config.VarzUser, "User")
}

type MyConfig struct {
	Config
	CustomProperty string
}

func TestReadingMyConfigFromJsonFile(t *testing.T) {
	file, err := ioutil.TempFile("", "config")
	defer func() {
		os.Remove(file.Name())
	}()
	assert.NoError(t, err)
	_, err = file.Write([]byte(`{"VarzUser":"User", "CustomProperty":"CustomValue"}`))
	assert.NoError(t, err)

	err = file.Close()
	assert.NoError(t, err)

	config := &MyConfig{}
	err = ReadConfigInto(config, file.Name())
	assert.NoError(t, err)

	assert.Equal(t, config.VarzUser, "User")
	assert.Equal(t, config.CustomProperty, "CustomValue")
}

func TestReturnsErrorIfFileNotFound(t *testing.T) {
	config := &Config{}
	err := ReadConfigInto(config, "/foo/config.json")
	assert.Error(t, err)
}

func TestReturnsErrorIfInvalidJson(t *testing.T) {
	file, err := ioutil.TempFile("", "config")
	defer func() {
		os.Remove(file.Name())
	}()
	assert.NoError(t, err)
	_, err = file.Write([]byte(`NotJson`))
	assert.NoError(t, err)

	err = file.Close()
	assert.NoError(t, err)

	config := &Config{}
	err = ReadConfigInto(config, file.Name())
	assert.Error(t, err)
}

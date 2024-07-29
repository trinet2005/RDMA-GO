package RDMAGO

import (
	"encoding/json"
	"os"
)

type Config struct {
	Mode       string `json:"mode"`
	Port       string `json:"port"`
	Address    string `json:"address"`
	Debug      bool   `json:"debug"`
	MrSize     int    `json:"mr_size"`
	DeviceName string `json:"device_name"`
	FileName   string `json:"file_name"`
}

// LoadConfig 加载配置文件
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

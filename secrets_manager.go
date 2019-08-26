package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// Load secret from store
func loadSecret(secretArn string) (map[string]interface{}, error) {
	svc := secretsmanager.New(session.Must(session.NewSession()))
	result, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{SecretId: aws.String(secretArn), VersionStage: aws.String("AWSCURRENT")})
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve secret %s: %s", secretArn, err)
	}

	var config map[string]interface{}
	err = json.Unmarshal([]byte(*result.SecretString), &config)
	if err != nil {
		log.Fatal("Can't unmarshal secret (expecting JSON payload): ", err)
	}

	for k, v := range config {
		if strings.HasPrefix(v.(string), "{") {
			o := make(map[string]interface{})
			err := json.Unmarshal([]byte(v.(string)), &o)
			if err != nil {
				panic(err)
			}
			config[k] = o
		}
	}

	return config, nil
}

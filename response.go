package main

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type response interface {
	String() string
}

type errorResponse struct {
	response
	Message string `json:"error"`
}

func (r *errorResponse) String() string {
	resp, err := json.Marshal(r)

	if err != nil {
		logrus.Fatal(err)
	}

	return string(resp)
}

type onCallUserResponse struct {
	response
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (r *onCallUserResponse) String() string {
	resp, err := json.Marshal(r)

	if err != nil {
		logrus.Fatal(err)
	}

	return string(resp)
}

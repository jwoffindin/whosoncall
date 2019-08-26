package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/kofalt/go-memoize"
	"github.com/sirupsen/logrus"
)

const (
	// Name of environment variable that secret ARN is being passed in as.
	secretsArnEnvironmentName = "PAGERDUTY_API_TOKEN"

	// The secret return by secrets manager is a JSON object. This is the top-level
	// map key holding the PagerDuty API key.
	pagerDutyAPIKeyField = "PagerDutyApiKey"
)

var (
	// Regexp used to validated schedule id passed in requests.
	validSchedule = regexp.MustCompile("^[A-Z0-9]{4,10}$").MatchString

	// We cache secret lookups and PagerDuty on-call responses. We're not
	// expecting much traffic and the API GW should apply fairly conservative
	// rate limits so not entirely necessary.
	cache = memoize.NewMemoizer(90*time.Second, 10*time.Minute)
)

// HandlerWithLogging is our entrypoint for handling api gateway proxy requests. Requests
// are authenticated by APIGateway before we receive them.
func HandlerWithLogging(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var body []byte
	var resp response

	requestLogger := baseLogger().WithField("requestID", "Req"+req.RequestContext.RequestID)

	resp, err := handler(ctx, requestLogger, req)
	if err != nil {
		resp = &errorResponse{Message: err.Error()}
	}
	body, err = json.Marshal(resp)

	return events.APIGatewayProxyResponse{
		Body:       string(body) + "\n",
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
	}, err
}

// handler calls PagerDuty to determine "who's on call" for the request.
func handler(ctx context.Context, logger *logrus.Entry, req events.APIGatewayProxyRequest) (*onCallUserResponse, error) {
	var schedule = req.QueryStringParameters["schedule"]
	logger = logger.WithField("schedule", schedule)

	if !validSchedule(schedule) {
		logger.Error("Received invalid schedule")
		return nil, errors.New("Invalid schedule")
	}

	retVal, err, _ := cache.Memoize("schedule-"+schedule, func() (interface{}, error) {
		return whosOnCall(logger, schedule)
	})

	return retVal.(*onCallUserResponse), err
}

func whosOnCall(logger *logrus.Entry, schedule string) (*onCallUserResponse, error) {
	client, err := pagerdutyClient(logger)
	if err != nil {
		logger.Error(err)
		return nil, errors.New("Unable to connect to PagerDuty")
	}

	logger.Info("Calling pagerduty API")
	onCall, err := client.ListOnCalls(pagerduty.ListOnCallOptions{Includes: []string{"users"}, ScheduleIDs: []string{schedule}})
	if err != nil {
		logger.Error("Unable to request on-call person: ", err)
		return nil, errors.New("Unable to determine on-call person")
	}

	if len(onCall.OnCalls) < 1 {
		logger.Error("No users returned in request")
		return nil, errors.New("No on-call person")
	}

	user := onCall.OnCalls[0].User

	logger.Infof("Have oncall user %#v", user)

	return &onCallUserResponse{Name: user.Name, Email: user.Email}, nil
}

func pagerdutyClient(logger *logrus.Entry) (*pagerduty.Client, error) {
	cached, err, _ := cache.Memoize("clientKey", func() (interface{}, error) {
		return getAPIKey(logger)
	})

	if err != nil {
		return nil, fmt.Errorf("Unable to load secret from AWS SecretsManager. arn=%s, error=%s", secretsArnEnvironmentName, err)
	}

	authtoken, ok := cached.(string)
	if !ok {
		return nil, fmt.Errorf("Unable to extract auth token as string, got a %T", cached)
	}

	return pagerduty.NewClient(authtoken), nil
}

func getAPIKey(logger *logrus.Entry) (interface{}, error) {
	secretsArn, ok := os.LookupEnv(secretsArnEnvironmentName)
	if !ok {
		err := errors.New("Unable to find secrets manager key in environment")
		logger.WithField("EnvKey", secretsArnEnvironmentName).Error(err)
		return nil, err
	}

	logger.WithField("SecretARN", secretsArn).Info("Retrieving config from secrets manager")
	config, err := loadSecret(secretsArn)
	val, ok := config[pagerDutyAPIKeyField]
	if !ok {
		return nil, fmt.Errorf("Secrets manager config does not have key %q", pagerDutyAPIKeyField)
	}

	return val, err
}

func baseLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.DebugLevel)

	return logger
}

func main() {
	lambda.Start(HandlerWithLogging)
}

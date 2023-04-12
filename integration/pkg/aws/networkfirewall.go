package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	nfwTypes "github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	smithytime "github.com/aws/smithy-go/time"
	smithywaiter "github.com/aws/smithy-go/waiter"
	"github.com/jmespath/go-jmespath"
)

type byovpcNetworkFirewallApi interface {
	CreateRuleGroup(ctx context.Context, params *networkfirewall.CreateRuleGroupInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.CreateRuleGroupOutput, error)
	DescribeRuleGroup(ctx context.Context, params *networkfirewall.DescribeRuleGroupInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeRuleGroupOutput, error)
	CreateFirewallPolicy(ctx context.Context, params *networkfirewall.CreateFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.CreateFirewallPolicyOutput, error)
	DescribeFirewallPolicy(ctx context.Context, params *networkfirewall.DescribeFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallPolicyOutput, error)
	CreateFirewall(ctx context.Context, params *networkfirewall.CreateFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.CreateFirewallOutput, error)
	DescribeFirewall(ctx context.Context, params *networkfirewall.DescribeFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallOutput, error)

	DeleteFirewall(ctx context.Context, params *networkfirewall.DeleteFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteFirewallOutput, error)
	DeleteFirewallPolicy(ctx context.Context, params *networkfirewall.DeleteFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteFirewallPolicyOutput, error)
	DeleteRuleGroup(ctx context.Context, params *networkfirewall.DeleteRuleGroupInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteRuleGroupOutput, error)
}

// aws-sdk-go doesn't currently have waiters for firewalls, which is a shame because they take so long to be created
// and deleted. The following code mirrors existing implementations from AWS, such as the DescribeNatGatewayWaiter

// DescribeFirewallAPIClient is a client that implements the DescribeFirewall
// operation.
type DescribeFirewallAPIClient interface {
	DescribeFirewall(ctx context.Context, params *networkfirewall.DescribeFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallOutput, error)
}

// FirewallReadyWaiterOptions are waiter options for FirewallReadyWaiter
type FirewallReadyWaiterOptions struct {
	// Set of options to modify how an operation is invoked. These apply to all
	// operations invoked for this client. Use functional options on operation call to
	// modify this list for per operation behavior.
	APIOptions []func(*middleware.Stack) error

	// MinDelay is the minimum amount of time to delay between retries. If unset,
	// FirewallReadyWaiter will use default minimum delay of 15 seconds. Note
	// that MinDelay must resolve to a value lesser than or equal to the MaxDelay.
	MinDelay time.Duration

	// MaxDelay is the maximum amount of time to delay between retries. If unset or set
	// to zero, FirewallReadyWaiter will use default max delay of 120 seconds.
	// Note that MaxDelay must resolve to value greater than or equal to the MinDelay.
	MaxDelay time.Duration

	// LogWaitAttempts is used to enable logging for waiter retry attempts
	LogWaitAttempts bool

	// Retryable is function that can be used to override the service defined
	// waiter-behavior based on operation output, or returned error. This function is
	// used by the waiter to decide if a state is retryable or a terminal state. By
	// default service-modeled logic will populate this option. This option can thus be
	// used to define a custom waiter state with fall-back to service-modeled waiter
	// state mutators.The function returns an error in case of a failure state. In case
	// of retry state, this function returns a bool value of true and nil error, while
	// in case of success it returns a bool value of false and nil error.
	Retryable func(context.Context, *networkfirewall.DescribeFirewallInput, *networkfirewall.DescribeFirewallOutput, error) (bool, error)
}

// FirewallReadyWaiter defines the waiters for FirewallReady
type FirewallReadyWaiter struct {
	client DescribeFirewallAPIClient

	options FirewallReadyWaiterOptions
}

// NewFirewallReadyWaiter constructs a FirewallReadyWaiter.
func NewFirewallReadyWaiter(client DescribeFirewallAPIClient, optFns ...func(options *FirewallReadyWaiterOptions)) *FirewallReadyWaiter {
	options := FirewallReadyWaiterOptions{}
	options.MinDelay = 15 * time.Second
	options.MaxDelay = 120 * time.Second
	options.Retryable = firewallReadyStateRetryable

	for _, fn := range optFns {
		fn(&options)
	}
	return &FirewallReadyWaiter{
		client:  client,
		options: options,
	}
}

// Wait calls the waiter function for FirewallReady waiter. The maxWaitDur is
// the maximum wait duration the waiter will wait. The maxWaitDur is required and
// must be greater than zero.
func (w *FirewallReadyWaiter) Wait(ctx context.Context, params *networkfirewall.DescribeFirewallInput, maxWaitDur time.Duration, optFns ...func(options *FirewallReadyWaiterOptions)) error {
	_, err := w.WaitForOutput(ctx, params, maxWaitDur, optFns...)
	return err
}

// WaitForOutput calls the waiter function for FirewallReady waiter and
// returns the output of the successful operation. The maxWaitDur is the maximum
// wait duration the waiter will wait. The maxWaitDur is required and must be
// greater than zero.
func (w *FirewallReadyWaiter) WaitForOutput(ctx context.Context, params *networkfirewall.DescribeFirewallInput, maxWaitDur time.Duration, optFns ...func(options *FirewallReadyWaiterOptions)) (*networkfirewall.DescribeFirewallOutput, error) {
	if maxWaitDur <= 0 {
		return nil, fmt.Errorf("maximum wait time for waiter must be greater than zero")
	}

	options := w.options
	for _, fn := range optFns {
		fn(&options)
	}

	if options.MaxDelay <= 0 {
		options.MaxDelay = 120 * time.Second
	}

	if options.MinDelay > options.MaxDelay {
		return nil, fmt.Errorf("minimum waiter delay %v must be lesser than or equal to maximum waiter delay of %v", options.MinDelay, options.MaxDelay)
	}

	ctx, cancelFn := context.WithTimeout(ctx, maxWaitDur)
	defer cancelFn()

	logger := smithywaiter.Logger{}
	remainingTime := maxWaitDur

	var attempt int64
	for {

		attempt++
		apiOptions := options.APIOptions
		start := time.Now()

		if options.LogWaitAttempts {
			logger.Attempt = attempt
			apiOptions = append([]func(*middleware.Stack) error{}, options.APIOptions...)
			apiOptions = append(apiOptions, logger.AddLogger)
		}

		out, err := w.client.DescribeFirewall(ctx, params, func(o *networkfirewall.Options) {
			o.APIOptions = append(o.APIOptions, apiOptions...)
		})

		retryable, err := options.Retryable(ctx, params, out, err)
		if err != nil {
			return nil, err
		}
		if !retryable {
			return out, nil
		}

		remainingTime -= time.Since(start)
		if remainingTime < options.MinDelay || remainingTime <= 0 {
			break
		}

		// compute exponential backoff between waiter retries
		delay, err := smithywaiter.ComputeDelay(
			attempt, options.MinDelay, options.MaxDelay, remainingTime,
		)
		if err != nil {
			return nil, fmt.Errorf("error computing waiter delay, %w", err)
		}

		remainingTime -= delay
		// sleep for the delay amount before invoking a request
		if err := smithytime.SleepWithContext(ctx, delay); err != nil {
			return nil, fmt.Errorf("request cancelled while waiting, %w", err)
		}
	}
	return nil, fmt.Errorf("exceeded max wait time for FirewallReady waiter")
}

func firewallReadyStateRetryable(ctx context.Context, input *networkfirewall.DescribeFirewallInput, output *networkfirewall.DescribeFirewallOutput, err error) (bool, error) {

	if err == nil {
		pathValue, err := jmespath.Search("FirewallStatus.Status", output)
		if err != nil {
			return false, fmt.Errorf("error evaluating waiter state: %w", err)
		}

		expectedValue := nfwTypes.FirewallStatusValueReady
		value, ok := pathValue.(nfwTypes.FirewallStatusValue)
		if !ok {
			if pathValue == nil {
				return true, nil
			}
			return false, fmt.Errorf("waiter comparator expected string value got %T", pathValue)
		}

		log.Printf("status: %s", output.FirewallStatus.Status)
		if value == expectedValue {
			return false, nil
		}
	}

	if err != nil {
		var apiErr smithy.APIError
		ok := errors.As(err, &apiErr)
		if !ok {
			return false, fmt.Errorf("expected err to be of type smithy.APIError, got %w", err)
		}

		if "ResourceNotFoundException" == apiErr.ErrorCode() {
			return true, nil
		}
	}

	return true, nil
}

// FirewallDeletedWaiterOptions are waiter options for FirewallDeletedWaiter
type FirewallDeletedWaiterOptions struct {
	// Set of options to modify how an operation is invoked. These apply to all
	// operations invoked for this client. Use functional options on operation call to
	// modify this list for per operation behavior.
	APIOptions []func(*middleware.Stack) error

	// MinDelay is the minimum amount of time to delay between retries. If unset,
	// FirewallDeletedWaiter will use default minimum delay of 15 seconds. Note
	// that MinDelay must resolve to a value lesser than or equal to the MaxDelay.
	MinDelay time.Duration

	// MaxDelay is the maximum amount of time to delay between retries. If unset or set
	// to zero, FirewallDeletedWaiter will use default max delay of 120 seconds.
	// Note that MaxDelay must resolve to value greater than or equal to the MinDelay.
	MaxDelay time.Duration

	// LogWaitAttempts is used to enable logging for waiter retry attempts
	LogWaitAttempts bool

	// Retryable is function that can be used to override the service defined
	// waiter-behavior based on operation output, or returned error. This function is
	// used by the waiter to decide if a state is retryable or a terminal state. By
	// default service-modeled logic will populate this option. This option can thus be
	// used to define a custom waiter state with fall-back to service-modeled waiter
	// state mutators.The function returns an error in case of a failure state. In case
	// of retry state, this function returns a bool value of true and nil error, while
	// in case of success it returns a bool value of false and nil error.
	Retryable func(context.Context, *networkfirewall.DescribeFirewallInput, *networkfirewall.DescribeFirewallOutput, error) (bool, error)
}

// FirewallDeletedWaiter defines the waiters for FirewallDeleted
type FirewallDeletedWaiter struct {
	client DescribeFirewallAPIClient

	options FirewallDeletedWaiterOptions
}

// NewFirewallDeletedWaiter constructs a FirewallDeletedWaiter.
func NewFirewallDeletedWaiter(client DescribeFirewallAPIClient, optFns ...func(options *FirewallDeletedWaiterOptions)) *FirewallDeletedWaiter {
	options := FirewallDeletedWaiterOptions{}
	options.MinDelay = 15 * time.Second
	options.MaxDelay = 120 * time.Second
	options.Retryable = firewallDeletedStateRetryable

	for _, fn := range optFns {
		fn(&options)
	}
	return &FirewallDeletedWaiter{
		client:  client,
		options: options,
	}
}

// Wait calls the waiter function for FirewallDeleted waiter. The maxWaitDur is
// the maximum wait duration the waiter will wait. The maxWaitDur is required and
// must be greater than zero.
func (w *FirewallDeletedWaiter) Wait(ctx context.Context, params *networkfirewall.DescribeFirewallInput, maxWaitDur time.Duration, optFns ...func(options *FirewallDeletedWaiterOptions)) error {
	_, err := w.WaitForOutput(ctx, params, maxWaitDur, optFns...)
	return err
}

// WaitForOutput calls the waiter function for FirewallDeleted waiter and
// returns the output of the successful operation. The maxWaitDur is the maximum
// wait duration the waiter will wait. The maxWaitDur is required and must be
// greater than zero.
func (w *FirewallDeletedWaiter) WaitForOutput(ctx context.Context, params *networkfirewall.DescribeFirewallInput, maxWaitDur time.Duration, optFns ...func(options *FirewallDeletedWaiterOptions)) (*networkfirewall.DescribeFirewallOutput, error) {
	if maxWaitDur <= 0 {
		return nil, fmt.Errorf("maximum wait time for waiter must be greater than zero")
	}

	options := w.options
	for _, fn := range optFns {
		fn(&options)
	}

	if options.MaxDelay <= 0 {
		options.MaxDelay = 120 * time.Second
	}

	if options.MinDelay > options.MaxDelay {
		return nil, fmt.Errorf("minimum waiter delay %v must be lesser than or equal to maximum waiter delay of %v", options.MinDelay, options.MaxDelay)
	}

	ctx, cancelFn := context.WithTimeout(ctx, maxWaitDur)
	defer cancelFn()

	logger := smithywaiter.Logger{}
	remainingTime := maxWaitDur

	var attempt int64
	for {

		attempt++
		apiOptions := options.APIOptions
		start := time.Now()

		if options.LogWaitAttempts {
			logger.Attempt = attempt
			apiOptions = append([]func(*middleware.Stack) error{}, options.APIOptions...)
			apiOptions = append(apiOptions, logger.AddLogger)
		}

		out, err := w.client.DescribeFirewall(ctx, params, func(o *networkfirewall.Options) {
			o.APIOptions = append(o.APIOptions, apiOptions...)
		})

		retryable, err := options.Retryable(ctx, params, out, err)
		if err != nil {
			return nil, err
		}
		if !retryable {
			return out, nil
		}

		remainingTime -= time.Since(start)
		if remainingTime < options.MinDelay || remainingTime <= 0 {
			break
		}

		// compute exponential backoff between waiter retries
		delay, err := smithywaiter.ComputeDelay(
			attempt, options.MinDelay, options.MaxDelay, remainingTime,
		)
		if err != nil {
			return nil, fmt.Errorf("error computing waiter delay, %w", err)
		}

		remainingTime -= delay
		// sleep for the delay amount before invoking a request
		if err := smithytime.SleepWithContext(ctx, delay); err != nil {
			return nil, fmt.Errorf("request cancelled while waiting, %w", err)
		}
	}
	return nil, fmt.Errorf("exceeded max wait time for FirewallDeleted waiter")
}

func firewallDeletedStateRetryable(ctx context.Context, input *networkfirewall.DescribeFirewallInput, output *networkfirewall.DescribeFirewallOutput, err error) (bool, error) {

	if err == nil {
		pathValue, err := jmespath.Search("FirewallStatus.Status", output)
		if err != nil {
			return false, fmt.Errorf("error evaluating waiter state: %w", err)
		}

		expectedValue := nfwTypes.FirewallStatusValueDeleting
		value, ok := pathValue.(nfwTypes.FirewallStatusValue)
		if !ok {
			if pathValue == nil {
				return true, nil
			}
			return false, fmt.Errorf("waiter comparator expected string value got %T", pathValue)
		}

		log.Printf("status: %s", output.FirewallStatus.Status)
		if value == expectedValue {
			return true, nil
		}
	}

	if err != nil {
		var apiErr smithy.APIError
		ok := errors.As(err, &apiErr)
		if !ok {
			return false, fmt.Errorf("expected err to be of type smithy.APIError, got %w", err)
		}

		if "ResourceNotFoundException" == apiErr.ErrorCode() {
			return false, nil
		}
	}

	return true, nil
}

// DescribeFirewallPolicyAPIClient is a client that implements the DescribeFirewallPolicy
// operation.
type DescribeFirewallPolicyAPIClient interface {
	DescribeFirewallPolicy(ctx context.Context, params *networkfirewall.DescribeFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallPolicyOutput, error)
}

// FirewallPolicyDeletedWaiterOptions are waiter options for FirewallPolicyDeletedWaiter
type FirewallPolicyDeletedWaiterOptions struct {
	// Set of options to modify how an operation is invoked. These apply to all
	// operations invoked for this client. Use functional options on operation call to
	// modify this list for per operation behavior.
	APIOptions []func(*middleware.Stack) error

	// MinDelay is the minimum amount of time to delay between retries. If unset,
	// FirewallPolicyDeletedWaiter will use default minimum delay of 15 seconds. Note
	// that MinDelay must resolve to a value lesser than or equal to the MaxDelay.
	MinDelay time.Duration

	// MaxDelay is the maximum amount of time to delay between retries. If unset or set
	// to zero, FirewallPolicyDeletedWaiter will use default max delay of 120 seconds.
	// Note that MaxDelay must resolve to value greater than or equal to the MinDelay.
	MaxDelay time.Duration

	// LogWaitAttempts is used to enable logging for waiter retry attempts
	LogWaitAttempts bool

	// Retryable is function that can be used to override the service defined
	// waiter-behavior based on operation output, or returned error. This function is
	// used by the waiter to decide if a state is retryable or a terminal state. By
	// default service-modeled logic will populate this option. This option can thus be
	// used to define a custom waiter state with fall-back to service-modeled waiter
	// state mutators.The function returns an error in case of a failure state. In case
	// of retry state, this function returns a bool value of true and nil error, while
	// in case of success it returns a bool value of false and nil error.
	Retryable func(context.Context, *networkfirewall.DescribeFirewallPolicyInput, *networkfirewall.DescribeFirewallPolicyOutput, error) (bool, error)
}

// FirewallPolicyDeletedWaiter defines the waiters for FirewallPolicyDeleted
type FirewallPolicyDeletedWaiter struct {
	client DescribeFirewallPolicyAPIClient

	options FirewallPolicyDeletedWaiterOptions
}

// NewFirewallPolicyDeletedWaiter constructs a FirewallPolicyDeletedWaiter.
func NewFirewallPolicyDeletedWaiter(client DescribeFirewallPolicyAPIClient, optFns ...func(options *FirewallPolicyDeletedWaiterOptions)) *FirewallPolicyDeletedWaiter {
	options := FirewallPolicyDeletedWaiterOptions{}
	options.MinDelay = 15 * time.Second
	options.MaxDelay = 120 * time.Second
	options.Retryable = firewallPolicyDeletedStateRetryable

	for _, fn := range optFns {
		fn(&options)
	}
	return &FirewallPolicyDeletedWaiter{
		client:  client,
		options: options,
	}
}

// Wait calls the waiter function for FirewallPolicyDeleted waiter. The maxWaitDur is
// the maximum wait duration the waiter will wait. The maxWaitDur is required and
// must be greater than zero.
func (w *FirewallPolicyDeletedWaiter) Wait(ctx context.Context, params *networkfirewall.DescribeFirewallPolicyInput, maxWaitDur time.Duration, optFns ...func(options *FirewallPolicyDeletedWaiterOptions)) error {
	_, err := w.WaitForOutput(ctx, params, maxWaitDur, optFns...)
	return err
}

// WaitForOutput calls the waiter function for FirewallPolicyDeleted waiter and
// returns the output of the successful operation. The maxWaitDur is the maximum
// wait duration the waiter will wait. The maxWaitDur is required and must be
// greater than zero.
func (w *FirewallPolicyDeletedWaiter) WaitForOutput(ctx context.Context, params *networkfirewall.DescribeFirewallPolicyInput, maxWaitDur time.Duration, optFns ...func(options *FirewallPolicyDeletedWaiterOptions)) (*networkfirewall.DescribeFirewallPolicyOutput, error) {
	if maxWaitDur <= 0 {
		return nil, fmt.Errorf("maximum wait time for waiter must be greater than zero")
	}

	options := w.options
	for _, fn := range optFns {
		fn(&options)
	}

	if options.MaxDelay <= 0 {
		options.MaxDelay = 120 * time.Second
	}

	if options.MinDelay > options.MaxDelay {
		return nil, fmt.Errorf("minimum waiter delay %v must be lesser than or equal to maximum waiter delay of %v", options.MinDelay, options.MaxDelay)
	}

	ctx, cancelFn := context.WithTimeout(ctx, maxWaitDur)
	defer cancelFn()

	logger := smithywaiter.Logger{}
	remainingTime := maxWaitDur

	var attempt int64
	for {

		attempt++
		apiOptions := options.APIOptions
		start := time.Now()

		if options.LogWaitAttempts {
			logger.Attempt = attempt
			apiOptions = append([]func(*middleware.Stack) error{}, options.APIOptions...)
			apiOptions = append(apiOptions, logger.AddLogger)
		}

		out, err := w.client.DescribeFirewallPolicy(ctx, params, func(o *networkfirewall.Options) {
			o.APIOptions = append(o.APIOptions, apiOptions...)
		})

		retryable, err := options.Retryable(ctx, params, out, err)
		if err != nil {
			return nil, err
		}
		if !retryable {
			return out, nil
		}

		remainingTime -= time.Since(start)
		if remainingTime < options.MinDelay || remainingTime <= 0 {
			break
		}

		// compute exponential backoff between waiter retries
		delay, err := smithywaiter.ComputeDelay(
			attempt, options.MinDelay, options.MaxDelay, remainingTime,
		)
		if err != nil {
			return nil, fmt.Errorf("error computing waiter delay, %w", err)
		}

		remainingTime -= delay
		// sleep for the delay amount before invoking a request
		if err := smithytime.SleepWithContext(ctx, delay); err != nil {
			return nil, fmt.Errorf("request cancelled while waiting, %w", err)
		}
	}
	return nil, fmt.Errorf("exceeded max wait time for FirewallPolicyDeleted waiter")
}

func firewallPolicyDeletedStateRetryable(ctx context.Context, input *networkfirewall.DescribeFirewallPolicyInput, output *networkfirewall.DescribeFirewallPolicyOutput, err error) (bool, error) {

	if err == nil {
		pathValue, err := jmespath.Search("FirewallPolicyResponse.FirewallPolicyStatus", output)
		if err != nil {
			return false, fmt.Errorf("error evaluating waiter state: %w", err)
		}

		expectedValue := nfwTypes.ResourceStatusDeleting
		value, ok := pathValue.(nfwTypes.ResourceStatus)
		if !ok {
			if pathValue == nil {
				return true, nil
			}
			return false, fmt.Errorf("waiter comparator expected string value got %T", pathValue)
		}

		log.Printf("status: %s", output.FirewallPolicyResponse.FirewallPolicyStatus)
		if value == expectedValue {
			return true, nil
		}
	}

	if err != nil {
		var apiErr smithy.APIError
		ok := errors.As(err, &apiErr)
		if !ok {
			return false, fmt.Errorf("expected err to be of type smithy.APIError, got %w", err)
		}

		if "ResourceNotFoundException" == apiErr.ErrorCode() {
			return false, nil
		}
	}

	return true, nil
}

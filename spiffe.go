package vault_auth_spiffe

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/command/agentproxyshared/auth"
	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

type spiffeMethod struct {
	logger          hclog.Logger
	mountPath       string
	role            string
	audience        string
	credsFound      chan struct{}
	stopCh          chan struct{}
	doneCh          chan struct{}
	credSuccessGate chan struct{}
	once            *sync.Once
	latestToken     *atomic.Value
	ticker          *time.Ticker
}

// NewSPIFFEAuthMethod returns an implementation of Agent's auth.AuthMethod
// interface for SPIFFE auth.
func NewSPIFFEAuthMethod(conf *auth.AuthConfig) (auth.AuthMethod, error) {
	if conf == nil {
		return nil, errors.New("empty config")
	}
	if conf.Config == nil {
		return nil, errors.New("empty config data")
	}

	j := &spiffeMethod{
		logger:          conf.Logger,
		mountPath:       conf.MountPath,
		credsFound:      make(chan struct{}),
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
		credSuccessGate: make(chan struct{}),
		once:            new(sync.Once),
		latestToken:     new(atomic.Value),
	}
	j.latestToken.Store("")

	roleRaw, ok := conf.Config["role"]
	if !ok {
		return nil, errors.New("missing 'role' value")
	}
	j.role, ok = roleRaw.(string)
	if !ok {
		return nil, errors.New("could not convert 'role' config value to string")
	}

	audienceRaw, ok := conf.Config["audience"]
	if !ok {
		return nil, errors.New("missing 'audience' value")
	} else {
		j.audience, ok = audienceRaw.(string)
		if !ok {
			return nil, errors.New("could not convert 'audience' config value to string")
		}
	}

	if j.role == "" {
		return nil, errors.New("'role' value is empty")
	}
	if j.audience == "" {
		return nil, errors.New("'audience' value is empty")
	}

	j.ticker = time.NewTicker(10 * time.Second)

	go j.runWatcher()

	j.logger.Info("spiffe auth method created")

	return j, nil
}

func (j *spiffeMethod) Authenticate(_ context.Context, _ *api.Client) (string, http.Header, map[string]interface{}, error) {
	j.logger.Trace("beginning authentication")

	latestToken := j.latestToken.Load().(string)
	if latestToken == "" {
		return "", nil, nil, errors.New("spiffe jwt-svid is not available")
	}

	return fmt.Sprintf("%s/login", j.mountPath), nil, map[string]interface{}{
		"role": j.role,
		"jwt":  latestToken,
	}, nil
}

func (j *spiffeMethod) NewCreds() chan struct{} {
	return j.credsFound
}

func (j *spiffeMethod) CredSuccess() {
	j.once.Do(func() {
		close(j.credSuccessGate)
	})
}

func (j *spiffeMethod) Shutdown() {
	j.ticker.Stop()
	close(j.stopCh)
	<-j.doneCh
}

func (j *spiffeMethod) runWatcher() {
	defer close(j.doneCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-j.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	// Fetch the first one right away
	j.fetchToken(ctx)

	for {
		select {
		case <-j.stopCh:
			return
		case <-j.ticker.C:
			j.fetchToken(ctx)
		}
	}
}

func (j *spiffeMethod) fetchToken(ctx context.Context) {
	params := jwtsvid.Params{
		Audience: j.audience,
	}
	svid, err := workloadapi.FetchJWTSVID(ctx, params)
	if err != nil {
		j.logger.Error("failed to fetch jwt-svid", "error", err)
		return
	}

	latestToken := j.latestToken.Load().(string)
	if svid.Marshal() != latestToken {
		j.latestToken.Store(svid.Marshal())
		j.logger.Debug("new jwt-svid available")

		// Only signal for new creds if we are not in the initial startup phase.
		// The initial creds are handled by the first Authenticate call.
		select {
		case <-j.credSuccessGate:
			// already closed, so we are past the initial auth
			j.credsFound <- struct{}{}
		default:
			// not closed yet, do nothing
		}
	}
}

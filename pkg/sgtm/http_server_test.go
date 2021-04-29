package sgtm

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"moul.io/sgtm/pkg/sgtmpb"
)

// FIXME: gRPC test
// FIXME: auth test
// FIXME: flow test
// FIXME: test sitemap

func TestServiceStatus(t *testing.T) {
	svc, cleanup := TestingService(t)
	defer cleanup()
	require.NotNil(t, svc)
	logger := TestingLogger(t)
	ctx := context.Background()

	time.Sleep(time.Second)
	fmt.Println(svc, logger, ctx)
	apiPrefix := fmt.Sprintf("http://%s/api/v1/", svc.ServerListenerAddr())
	resp, err := http.Get(apiPrefix + "Status")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var status sgtmpb.Status_Response
	err = json.Unmarshal(body, &status)
	require.NoError(t, err)
	require.NotEmpty(t, status.Hostname)
	require.True(t, status.Uptime >= 1)
	require.True(t, status.EverythingIsOk)
	// fmt.Println(godev.PrettyJSONPB(&status))
}

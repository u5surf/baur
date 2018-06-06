package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	docker "docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"github.com/pkg/errors"
	"github.com/simplesurance/baur/log"
)

// Client is a docker client
type Client struct {
	clt      *docker.Client
	authData string
}

func base64AuthData(user, password string) (string, error) {
	ac := types.AuthConfig{
		Username: user,
		Password: password,
	}

	js, err := json.Marshal(ac)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(js), nil
}

// NewClient intializes a new docker client with the given docker registry
// authentication data.
// The following environment variables are respected:
// Use DOCKER_HOST to set the url to the docker server.
// Use DOCKER_API_VERSION to set the version of the API to reach, leave empty for latest.
// Use DOCKER_CERT_PATH to load the TLS certificates from.
// Use DOCKER_TLS_VERIFY to enable or disable TLS verification, off by default.
func NewClient(user, password string) (*Client, error) {
	clt, err := docker.NewEnvClient()
	if err != nil {
		return nil, err
	}

	authData, err := base64AuthData(user, password)
	if err != nil {
		return nil, err
	}

	return &Client{
		clt:      clt,
		authData: authData,
	}, nil
}

func serverRespIsErr(in []byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(in, &m)
	if err != nil {
		return err
	}

	if _, exist := m["error"]; exist {
		prettyErr, err := json.MarshalIndent(&m, "", "  ")
		if err != nil {
			return errors.New(fmt.Sprint(m))
		}

		return errors.New(string(prettyErr))
	}

	return nil
}

// Upload tags and uploads an image into a docker registry repository
func (c *Client) Upload(ctx context.Context, image, dest string) (string, error) {
	err := c.clt.ImageTag(ctx, image, dest)
	if err != nil {
		return "", errors.Wrapf(err, "tagging image failed")
	}

	closer, err := c.clt.ImagePush(ctx, dest, types.ImagePushOptions{
		RegistryAuth: c.authData,
	})
	if err != nil {
		return "", errors.Wrapf(err, "pushing image failed")
	}

	defer closer.Close()

	r := bufio.NewReader(closer)
	for {
		status, err := r.ReadBytes('\n')

		log.Debugf("docker Upload of %s to %s, read server response: %q\n",
			image, dest, status)

		if err == io.EOF {
			break
		}

		if err := serverRespIsErr(status); err != nil {
			return "", errors.Wrapf(err, "pushing image failed")
		}
	}

	return dest, nil
}

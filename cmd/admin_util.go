package cmd

import "github.com/bak1an/artf/admin"

func initAdminClient(data string) (*admin.AdminClient, error) {
	client, err := admin.NewAdminClient(data)
	if err != nil {
		return nil, err
	}

	err = client.Ping()
	if err != nil {
		return nil, err
	}

	return client, nil
}

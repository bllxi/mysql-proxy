package testclient

import (
	"time"
)

type TestClient struct {
	db *DBX
}

func (t *TestClient) Run() error {
	t.db = new(DBX)
	if err := t.db.Open(); err != nil {
		return err
	}

	for {
		t.db.db.Ping()

		time.Sleep(time.Second)
	}

	return nil
}

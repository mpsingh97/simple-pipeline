package postgres

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	PrimaryPool    *pgxpool.Pool
	ReplicaPools   []*pgxpool.Pool
	currentReplica int
}

func (c *Client) GetReplicaPool() *pgxpool.Pool {
	if len(c.ReplicaPools) == 0 {
		log.Fatal("No replicas available")
	}
	replica := c.ReplicaPools[c.currentReplica]
	c.currentReplica = (c.currentReplica + 1) % len(c.ReplicaPools)
	return replica
}

func (c *Client) Close() {
	c.PrimaryPool.Close()
	for _, rp := range c.ReplicaPools {
		rp.Close()
	}
}

func New(primary_url string, replica_urls []string) (*Client, error) {
	client := &Client{}

	conf, err := pgxpool.ParseConfig(primary_url)
	if err != nil {
		return nil, err
	}
	conf.MaxConns = 60
	conf.MinConns = 10
	conf.MaxConnLifetime = 30 * time.Minute
	conf.MaxConnIdleTime = 5 * time.Minute

	primaryPool, err := pgxpool.NewWithConfig(context.Background(), conf)
	if err != nil {
		return nil, err
	}
	client.PrimaryPool = primaryPool

	for _, url := range replica_urls {
		conf, err := pgxpool.ParseConfig(primary_url)
		if err != nil {
			return nil, err
		}
		conf.MaxConns = 60
		conf.MinConns = 10
		conf.MaxConnLifetime = 30 * time.Minute
		conf.MaxConnIdleTime = 5 * time.Minute
		pool, err := pgxpool.NewWithConfig(context.Background(), conf)
		if err != nil {
			log.Printf("Unable to connect to replica database %s: %v\n", url, err)
		} else {
			client.ReplicaPools = append(client.ReplicaPools, pool)
		}
	}

	return client, nil
}

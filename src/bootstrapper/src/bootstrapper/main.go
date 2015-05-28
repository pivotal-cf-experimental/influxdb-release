package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"gopkg.in/pivotal-cf-experimental/influxdb.v0/client"
)

const mainShardSpace = "default"

func main() {
	influxUrl := flag.String("influxUrl", "http://localhost:8086", "influx endpoint")
	influxUser := flag.String("user", "root", "influx user")
	influxPassword := flag.String("password", "root", "influx user's password")
	databaseName := flag.String("database", "", "Database name to create")
	retention := flag.String("retention", "14d", "Retention duration")
	replication := flag.Uint("replication", 1, "Replication count")
	flag.Parse()

	if len(*databaseName) == 0 {
		fmt.Println("Database name is required")
		flag.Usage()
		os.Exit(1)
	}

	influxServerUrl, _ := url.Parse(*influxUrl)
	influxConfig := &client.ClientConfig{
		Host:     influxServerUrl.Host,
		IsSecure: influxServerUrl.Scheme == "https",
		Username: *influxUser,
		Password: *influxPassword,
		Database: *databaseName,
	}
	dbClient, err := client.NewClient(influxConfig)
	if err != nil {
		panic(fmt.Errorf("Error connecting to Influx DB: %v", err))
	}

	existsErr := fmt.Sprintf("Server returned (409): database %s exists", *databaseName)
	err = dbClient.CreateDatabase(*databaseName)
	if err != nil && err.Error() != existsErr {
		panic(fmt.Errorf("Error creating DB '%s': %v", *databaseName, err))
	}

	shardSpaces, err := dbClient.GetShardSpaces()
	if err != nil {
		panic(fmt.Errorf("Error loading shard spaces: %v", err))
	}
	var defaultSpace *client.ShardSpace
	for _, space := range shardSpaces {
		if space.Name == mainShardSpace {
			defaultSpace = space
			break
		}
	}

	if defaultSpace == nil {
		defaultSpace := &client.ShardSpace{
			Name:              mainShardSpace,
			RetentionPolicy:   *retention,
			Regex:             "/.*/",
			ReplicationFactor: uint32(*replication),
		}
		err = dbClient.CreateShardSpace(*databaseName, defaultSpace)
		if err != nil {
			panic(fmt.Errorf("Error creating default retention policy: %v", err))
		}
		return
	}

	defaultSpace.RetentionPolicy = *retention
	defaultSpace.ReplicationFactor = uint32(*replication)
	err = dbClient.UpdateShardSpace(*databaseName, defaultSpace.Name, defaultSpace)
	if err != nil {
		panic(fmt.Errorf("Error updating default retention policy: %v", err))
	}
}

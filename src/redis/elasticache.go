package main

import (
	"fmt"
	"strconv"
	"strings"

	paasElasticacheBrokerRedis "github.com/alphagov/paas-elasticache-broker/providers/redis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type RedisNode struct {
	CacheClusterName string
	NodeNumber       *int
	ReplicationGroup *elasticache.ReplicationGroup
	ServiceInstance  cfclient.ServiceInstance
	Space            cfclient.Space
	Organisation     cfclient.Org
}

func ListRedisNodes(
	serviceInstances []cfclient.ServiceInstance,
	spacesByGuid map[string]cfclient.Space,
	orgsByGuid map[string]cfclient.Org,
	elasticacheClient *elasticache.ElastiCache,
) (map[string]RedisNode, error) {
	redisNodes := map[string]RedisNode{}
	for _, serviceInstance := range serviceInstances {
		replicationGroupName := paasElasticacheBrokerRedis.GenerateReplicationGroupName(serviceInstance.Guid)
		replicationGroup, err := getReplicationGroup(replicationGroupName, elasticacheClient)
		if err != nil {
			return nil, err
		}

		for _, cacheClusterName := range replicationGroup.MemberClusters {
			space := spacesByGuid[serviceInstance.SpaceGuid]
			redisNodes[*cacheClusterName] = RedisNode{
				CacheClusterName: *cacheClusterName,
				NodeNumber:       getNodeNumberFromCacheClusterName(*cacheClusterName),
				ReplicationGroup: replicationGroup,
				ServiceInstance:  serviceInstance,
				Space:            space,
				Organisation:     orgsByGuid[space.OrganizationGuid],
			}
		}
	}
	return redisNodes, nil
}

func getReplicationGroup(name string, elasticacheClient *elasticache.ElastiCache) (*elasticache.ReplicationGroup, error) {
	replicationGroupOutput, err := elasticacheClient.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: aws.String(name),
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching replication group '%v' from elasticache: %v", name, err)
	}
	if len(replicationGroupOutput.ReplicationGroups) != 1 {
		return nil, fmt.Errorf("got %d results fetching replication group '%s' from elasticache but expected 1 result", len(replicationGroupOutput.ReplicationGroups), name)
	}
	return replicationGroupOutput.ReplicationGroups[0], nil
}

func getNodeNumberFromCacheClusterName(name string) *int {
	segments := strings.Split(name, "-")
	nodeNumber, err := strconv.Atoi(segments[2])
	if err != nil {
		return nil
	}
	return &nodeNumber
}

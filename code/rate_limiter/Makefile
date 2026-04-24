CLUSTER_DIR := /tmp/redis-cluster
CLUSTER_PORTS := 7001 7002 7003 7004 7005 7006

redis-cluster-start:
	@for port in $(CLUSTER_PORTS); do \
		mkdir -p $(CLUSTER_DIR)/$$port; \
		redis-server \
			--port $$port \
			--cluster-enabled yes \
			--cluster-config-file $(CLUSTER_DIR)/$$port/nodes.conf \
			--cluster-node-timeout 5000 \
			--daemonize yes \
			--logfile $(CLUSTER_DIR)/$$port/redis.log \
			--dir $(CLUSTER_DIR)/$$port; \
	done
	@sleep 1
	redis-cli --cluster create \
		127.0.0.1:7001 127.0.0.1:7002 127.0.0.1:7003 \
		127.0.0.1:7004 127.0.0.1:7005 127.0.0.1:7006 \
		--cluster-replicas 1 --cluster-yes

redis-cluster-stop:
	@for port in $(CLUSTER_PORTS); do \
		redis-cli -p $$port shutdown nosave 2>/dev/null || true; \
	done
	@rm -rf $(CLUSTER_DIR)


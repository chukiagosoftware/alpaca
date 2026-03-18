# Makefile — place in alpaca/ root

STACK         := dev
INDEX_DIR     := pulumiVectorIndex
ENDPOINT_DIR  := pulumiVectorEndpoint
CONFIG_YAML   := config.yaml
IMPORT_SCRIPT := vertex/tools/importBQToIndex.sh

.PHONY: VertexIndexEndpoint

VertexIndexEndpoint:
	@echo "==> Step 1: pulumi up for VectorIndex"
	cd $(INDEX_DIR) && pulumi up --yes --stack $(STACK)

	@echo "==> Step 2: Read index ID from stack output"
	$(eval INDEX_ID := $(shell cd $(INDEX_DIR) && pulumi stack output indexId --stack $(STACK)))
	@echo "    Index ID: $(INDEX_ID)"

	@echo "==> Step 3: Import BigQuery data into index"
	VERTEX_INDEX_ID=$(INDEX_ID) bash $(IMPORT_SCRIPT)

	@echo "==> Step 4: Update config.yaml with new index_id"
	sed -i.bak 's/^index_id:.*/index_id: $(INDEX_ID)/' $(CONFIG_YAML)

	@echo "==> Step 5: Update pulumiVectorEndpoint stack config"
	cd $(ENDPOINT_DIR) && pulumi config set --stack $(STACK) pulumiVertexSearch:indexID $(INDEX_ID)

	@echo "==> Step 6: pulumi up for VectorEndpoint"
	cd $(ENDPOINT_DIR) && pulumi up --yes --stack $(STACK)

	@echo "==> Done! Index ID $(INDEX_ID) deployed end to end."
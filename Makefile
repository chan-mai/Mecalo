.PHONY: gen lint test test-cockroach dev-backend dev-frontend build models

# TypeSpec → OpenAPI → Go / TypeScript
gen:
	pnpm --filter @mecalo/api build
	cd backend && go generate ./...
	pnpm --filter @mecalo/frontend gen:api

lint:
	cd backend && go vet ./... && go tool golangci-lint run ./...
	pnpm --filter @mecalo/frontend lint
	pnpm --filter @mecalo/frontend typecheck

test:
	cd backend && go test ./...

# MECALO_TEST_POSTGRES_DSN の CockroachDB でも契約テストを実行
test-cockroach:
	cd backend && MECALO_TEST_POSTGRES_DSN=$${MECALO_TEST_POSTGRES_DSN:-postgresql://root@localhost:26257/defaultdb?sslmode=disable} go test ./internal/app/...

dev-backend:
	cd backend && go run ./cmd/mecalo serve

dev-frontend:
	pnpm --filter @mecalo/frontend dev

build:
	cd backend && go build -o bin/mecalo ./cmd/mecalo
	pnpm --filter @mecalo/frontend build

# assets/blender/の各.blendのコレクションをglTFへ書き出し
BLENDER ?= blender
models:
	$(BLENDER) -b assets/blender/foods.blend --python-expr "import bpy; [o.select_set(o.name in bpy.data.collections['Foods'].objects) for o in bpy.context.scene.objects]; bpy.ops.export_scene.gltf(filepath='frontend/public/models/foods.glb', export_format='GLB', use_selection=True, export_apply=True)"
	$(BLENDER) -b assets/blender/pfc.blend --python-expr "import bpy; [o.select_set(o.name in bpy.data.collections['Pfc'].objects) for o in bpy.context.scene.objects]; bpy.ops.export_scene.gltf(filepath='frontend/public/models/pfc.glb', export_format='GLB', use_selection=True, export_apply=True)"

#!/bin/bash

echo "=== Comparación REST vs GraphQL ==="

# Crear tarea
echo "Creando tarea de ejemplo..."
TASK_ID=$(curl -s -X POST http://localhost:8082/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Tarea comparación","description":"REST vs GraphQL"}' \
  | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

echo "ID de tarea: $TASK_ID"

# REST
echo -e "\n--- REST ---"
curl -s http://localhost:8082/v1/tasks -o rest_list.json
curl -s http://localhost:8082/v1/tasks/$TASK_ID -o rest_detail.json
echo "Lista: $(wc -c < rest_list.json) bytes"
echo "Detalle: $(wc -c < rest_detail.json) bytes"

# GraphQL
echo -e "\n--- GraphQL ---"
curl -s -X POST http://localhost:8090/query \
  -H "Content-Type: application/json" \
  -d '{"query":"{ tasks { id title done } }"}' \
  -o gql_list.json

curl -s -X POST http://localhost:8090/query \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"query(\$id:ID!){ task(id:\$id) { id title description done createdAt } }\",\"variables\":{\"id\":\"$TASK_ID\"}}" \
  -o gql_detail.json

echo "Lista: $(wc -c < gql_list.json) bytes"
echo "Detalle: $(wc -c < gql_detail.json) bytes"

# Resultados
echo -e "\n=== RESULTADOS ==="
echo "REST Lista: $(wc -c < rest_list.json) bytes"
echo "GraphQL Lista: $(wc -c < gql_list.json) bytes"
echo "Ahorro: $(( $(wc -c < rest_list.json) - $(wc -c < gql_list.json) )) bytes"
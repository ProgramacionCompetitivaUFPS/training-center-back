-- +goose Up

-- Envíos hechos con el editor de bloques (Blockly): se juzgan como python310 con el código
-- generado, y además se guarda el espacio de trabajo (XML) junto al fuente para poder
-- mostrar los bloques en el detalle y recuperarlos en el editor. NULL = envío normal.
ALTER TABLE submissions ADD COLUMN workspace_xml_path TEXT;

-- +goose Down

ALTER TABLE submissions DROP COLUMN IF EXISTS workspace_xml_path;

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package migrations

import (
	"context"

	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

const syncPostgresIDSequencesSQL = `
DO $$
DECLARE
	seq RECORD;
BEGIN
	FOR seq IN
		SELECT
			n.nspname AS schema_name,
			c.relname AS table_name,
			a.attname AS column_name,
			pg_get_serial_sequence(format('%I.%I', n.nspname, c.relname), a.attname) AS sequence_name
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid
		WHERE c.relkind = 'r'
			AND a.attnum > 0
			AND NOT a.attisdropped
			AND a.attname = 'id'
			AND n.nspname = current_schema()
	LOOP
		IF seq.sequence_name IS NULL THEN
			CONTINUE;
		END IF;

		EXECUTE format(
			'SELECT setval(%L, COALESCE(max_id, 1), max_id IS NOT NULL) FROM (SELECT MAX(%I) AS max_id FROM %I.%I) seeded',
			seq.sequence_name,
			seq.column_name,
			seq.schema_name,
			seq.table_name
		);
	END LOOP;
END $$;`

func syncPostgresIDSequences(ctx context.Context, x *xorm.Engine) error {
	if x.Dialect().URI().DBType != schemas.POSTGRES {
		return nil
	}
	_, err := x.Context(ctx).Exec(syncPostgresIDSequencesSQL)
	return err
}

func syncPostgresSeededSequences(ctx context.Context, x *xorm.Engine) error {
	return syncPostgresIDSequences(ctx, x)
}

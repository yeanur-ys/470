// nextGENjournalism graph schema
// Nodes are created/updated by the CDC pipeline (Debezium -> Kafka -> graph-sync worker),
// mirroring the Postgres `articles` and `users` tables (Section 2.1, "Dual-Database Synergy").

CREATE CONSTRAINT article_id_unique IF NOT EXISTS
FOR (a:Article) REQUIRE a.id IS UNIQUE;

CREATE CONSTRAINT journalist_id_unique IF NOT EXISTS
FOR (j:Journalist) REQUIRE j.id IS UNIQUE;

CREATE CONSTRAINT tag_name_unique IF NOT EXISTS
FOR (t:Tag) REQUIRE t.name IS UNIQUE;

// Article lineage: directional chain of related stories (FR-4 Sequence Stitching)
// (:Article)-[:SEQUENCE_OF]->(:Article)

// Authorship
// (:Journalist)-[:AUTHORED]->(:Article)

// Claim tagging used for cross-tag auditor consensus (FR-7)
// (:Article)-[:HAS_TAG]->(:Tag)

CREATE INDEX article_corruption_idx IF NOT EXISTS FOR (a:Article) ON (a.corruptionFactor);
CREATE INDEX article_readership_idx IF NOT EXISTS FOR (a:Article) ON (a.readershipVolume);

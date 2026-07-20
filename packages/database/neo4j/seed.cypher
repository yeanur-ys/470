// Mirrors packages/database/postgres/seed.sql directly into the graph.
// Run this against Neo4j to see LineageGraph populated immediately, without
// needing the full Debezium -> Kafka -> CDC-sync pipeline running first.
// (In a real deployment, this data flows in automatically once an article
// is created in Postgres — this file exists purely so you can see the
// graph/cluster-legend/era-legend/dispute-pulse features without also
// standing up Kafka + Debezium + the Python worker just to try them.)
//
// clusterId assignments mirror three Louvain-style groupings the real
// Python worker would likely find given the SEQUENCE_OF chains below:
//   0 = the budget story chain (a1-a4)
//   1 = the transit story chain (a5-a6)
//   2 = the layoffs story chain + retraction (b1-b4)

MERGE (amara:Journalist {id: '11111111-1111-1111-1111-111111111111'})
MERGE (devon:Journalist {id: '22222222-2222-2222-2222-222222222222'})

MERGE (a1:Article {id: 'a0000000-0000-0000-0000-000000000001'})
SET a1.title = 'City Budget Shows $40M Shortfall', a1.readershipVolume = 5000,
    a1.corruptionFactor = 0.0, a1.isRetracted = false, a1.clusterId = 0,
    a1.createdAt = toString(datetime() - duration({days: 140}))
MERGE (amara)-[:AUTHORED]->(a1)

MERGE (a2:Article {id: 'a0000000-0000-0000-0000-000000000002'})
SET a2.title = 'Budget Shortfall Traced to Pension Fund', a2.readershipVolume = 3000,
    a2.corruptionFactor = 0.0, a2.isRetracted = false, a2.clusterId = 0,
    a2.createdAt = toString(datetime() - duration({days: 95}))
MERGE (amara)-[:AUTHORED]->(a2)
MERGE (a2)-[:SEQUENCE_OF]->(a1)

MERGE (a3:Article {id: 'a0000000-0000-0000-0000-000000000003'})
SET a3.title = 'Officials Deny Pension Mismanagement', a3.readershipVolume = 1200,
    a3.corruptionFactor = 1.0, a3.isRetracted = false, a3.clusterId = 0,
    a3.createdAt = toString(datetime() - duration({days: 9}))
MERGE (amara)-[:AUTHORED]->(a3)
MERGE (a3)-[:SEQUENCE_OF]->(a2)

MERGE (a4:Article {id: 'a0000000-0000-0000-0000-000000000004'})
SET a4.title = 'Follow-up: New Documents Surface', a4.readershipVolume = 800,
    a4.corruptionFactor = 0.0, a4.isRetracted = false, a4.clusterId = 0,
    a4.createdAt = toString(datetime() - duration({days: 2}))
MERGE (amara)-[:AUTHORED]->(a4)
MERGE (a4)-[:SEQUENCE_OF]->(a3)

MERGE (a5:Article {id: 'a0000000-0000-0000-0000-000000000005'})
SET a5.title = 'New Transit Line Opens Downtown', a5.readershipVolume = 10000,
    a5.corruptionFactor = 0.0, a5.isRetracted = false, a5.clusterId = 1,
    a5.createdAt = toString(datetime() - duration({days: 5}))
MERGE (amara)-[:AUTHORED]->(a5)

MERGE (a6:Article {id: 'a0000000-0000-0000-0000-000000000006'})
SET a6.title = 'Transit Ridership Exceeds Projections', a6.readershipVolume = 2000,
    a6.corruptionFactor = 0.0, a6.isRetracted = false, a6.clusterId = 1,
    a6.createdAt = toString(datetime() - duration({days: 1}))
MERGE (amara)-[:AUTHORED]->(a6)
MERGE (a6)-[:SEQUENCE_OF]->(a5)

MERGE (b1:Article {id: 'b0000000-0000-0000-0000-000000000001'})
SET b1.title = 'Tech Layoffs Hit Regional Startups', b1.readershipVolume = 1500,
    b1.corruptionFactor = 0.5, b1.isRetracted = false, b1.clusterId = 2,
    b1.createdAt = toString(datetime() - duration({days: 220}))
MERGE (devon)-[:AUTHORED]->(b1)

MERGE (b2:Article {id: 'b0000000-0000-0000-0000-000000000002'})
SET b2.title = 'Startup Founders Push Back on Layoff Report', b2.readershipVolume = 900,
    b2.corruptionFactor = 1.0, b2.isRetracted = false, b2.clusterId = 2,
    b2.createdAt = toString(datetime() - duration({days: 110}))
MERGE (devon)-[:AUTHORED]->(b2)
MERGE (b2)-[:SEQUENCE_OF]->(b1)

MERGE (b3:Article {id: 'b0000000-0000-0000-0000-000000000003'})
SET b3.title = 'Correction: Layoff Numbers Revised', b3.readershipVolume = 600,
    b3.corruptionFactor = 0.0, b3.isRetracted = false, b3.clusterId = 2,
    b3.createdAt = toString(datetime() - duration({days: 20}))
MERGE (devon)-[:AUTHORED]->(b3)
MERGE (b3)-[:SEQUENCE_OF]->(b2)

MERGE (b4:Article {id: 'b0000000-0000-0000-0000-000000000004'})
SET b4.title = '[retracted]', b4.readershipVolume = 400,
    b4.corruptionFactor = 1.0, b4.isRetracted = true, b4.clusterId = 2,
    b4.createdAt = toString(datetime() - duration({days: 260}))
MERGE (devon)-[:AUTHORED]->(b4)

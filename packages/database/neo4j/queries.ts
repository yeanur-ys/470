// Precompiled Cypher queries shared by any Node/TS process that talks to Neo4j
// (the graph-sync worker, or server-side data loaders in the frontend).

export const UPSERT_ARTICLE = `
  MERGE (a:Article {id: $id})
  SET a.title = $title,
      a.readershipVolume = $readershipVolume,
      a.corruptionFactor = $corruptionFactor,
      a.isRetracted = $isRetracted
  WITH a
  MATCH (j:Journalist {id: $journalistId})
  MERGE (j)-[:AUTHORED]->(a)
`;

export const LINK_SEQUENCE = `
  MATCH (parent:Article {id: $parentId})
  MATCH (child:Article {id: $childId})
  MERGE (child)-[:SEQUENCE_OF]->(parent)
`;

export const TAG_ARTICLE = `
  MATCH (a:Article {id: $articleId})
  MERGE (t:Tag {name: $tagName})
  MERGE (a)-[:HAS_TAG]->(t)
`;

export const GET_LINEAGE = `
  MATCH path = (a:Article {id: $rootId})-[:SEQUENCE_OF*0..]->(ancestor:Article)
  RETURN path
`;

export const GET_JOURNALIST_GRAPH = `
  MATCH (j:Journalist {id: $journalistId})-[:AUTHORED]->(a:Article)
  OPTIONAL MATCH (a)-[:SEQUENCE_OF]->(parent:Article)
  RETURN a, parent
  ORDER BY a.createdAt DESC
`;

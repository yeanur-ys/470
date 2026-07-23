from __future__ import annotations

from collections.abc import Iterable

import networkx as nx
from community import best_partition

# Lineage (SEQUENCE_OF) is the semantically meaningful relationship, so it
# carries more weight than a shared topic tag when Louvain decides where a
# community boundary falls. Two articles in the same story chain belong
# together more strongly than two that merely share a category.
SEQUENCE_WEIGHT = 3.0
TOPIC_WEIGHT = 1.0


def compute_louvain_clusters(
    edges: Iterable[tuple[str, str]],
    topic_edges: Iterable[tuple[str, str]] = (),
    all_nodes: Iterable[str] = (),
) -> dict[str, int]:
    """Partitions the article graph into communities for semantic zooming (F-07).

    Two changes from the original, both of which caused nodes to silently
    receive no cluster at all:

    1. `all_nodes` is added explicitly. `nx.Graph.add_edges_from` only creates
       nodes that appear in some edge, so any article without a parent and
       without a shared tag — which is every brand-new standalone story — was
       absent from the partition entirely and got no `clusterId`. Those nodes
       then fell into `useSemanticZoom`'s `attrs.clusterId ?? node` fallback,
       where each became its own singleton "cluster" and was therefore treated
       as its own hub, so it never collapsed under semantic zoom.

    2. `topic_edges` (shared HAS_TAG category) are folded in alongside lineage.
       SEQUENCE_OF alone is a forest — every article has at most one parent —
       and Louvain over a forest just recovers the individual trees, which is
       not community detection in any useful sense. Topic edges are what make
       the modularity optimisation meaningful.

    Isolated nodes each end up in their own community, which is the correct
    Louvain result for them.
    """
    graph = nx.Graph()

    graph.add_nodes_from(all_nodes)
    for source, target in edges:
        if source and target:
            graph.add_edge(source, target, weight=SEQUENCE_WEIGHT)
    for source, target in topic_edges:
        if not source or not target or source == target:
            continue
        if graph.has_edge(source, target):
            # A pair that is both a lineage link and a topic link is a stronger
            # signal than either alone, so the weights add rather than the
            # topic edge overwriting the lineage weight.
            graph[source][target]["weight"] += TOPIC_WEIGHT
        else:
            graph.add_edge(source, target, weight=TOPIC_WEIGHT)

    if graph.number_of_nodes() == 0:
        return {}

    # random_state pins the partition so cluster ids stay stable across polls.
    # Without it Louvain's randomised node ordering renumbers communities on
    # every run, and the frontend's cluster colours and legend would reshuffle
    # every 30 seconds on unchanged data.
    return best_partition(graph, weight="weight", random_state=42)

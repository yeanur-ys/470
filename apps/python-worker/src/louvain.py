from __future__ import annotations

from collections.abc import Iterable

import networkx as nx
from community import best_partition


def compute_louvain_clusters(edges: Iterable[tuple[str, str]]) -> dict[str, int]:
    graph = nx.Graph()
    graph.add_edges_from(edges)
    if graph.number_of_nodes() == 0:
        return {}
    return best_partition(graph)

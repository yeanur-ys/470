from __future__ import annotations

import time
import traceback

from config import load_config
from graph_store import (
    fetch_all_article_ids,
    fetch_edges,
    fetch_topic_edges,
    get_driver,
    write_clusters,
)
from louvain import compute_louvain_clusters


def run() -> None:
    config = load_config()
    driver = get_driver(config.neo4j_uri, config.neo4j_user, config.neo4j_password)

    try:
        while True:
            try:
                node_ids = fetch_all_article_ids(driver)
                edges = fetch_edges(driver)
                topic_edges = fetch_topic_edges(driver)
                clusters = compute_louvain_clusters(edges, topic_edges, node_ids)
                write_clusters(driver, clusters)

                distinct = len(set(clusters.values()))
                print(
                    f"worker heartbeat interval={config.poll_interval_seconds}s "
                    f"nodes={len(node_ids)} lineage_edges={len(edges)} "
                    f"topic_edges={len(topic_edges)} communities={distinct}",
                    flush=True,
                )
            except Exception:
                # A transient Neo4j blip must not kill the worker: the
                # container would restart, reconnect, and hit the same blip.
                # Log and wait for the next poll instead.
                print("worker poll failed:", flush=True)
                traceback.print_exc()

            time.sleep(config.poll_interval_seconds)
    finally:
        driver.close()


if __name__ == "__main__":
    run()

from __future__ import annotations

import time

from config import load_config
from graph_store import get_driver, fetch_edges, write_clusters
from louvain import compute_louvain_clusters


def run() -> None:
    config = load_config()
    driver = get_driver(config.neo4j_uri, config.neo4j_user, config.neo4j_password)

    try:
        while True:
            edges = fetch_edges(driver)
            clusters = compute_louvain_clusters(edges)
            write_clusters(driver, clusters)
            print(
                f"worker heartbeat interval={config.poll_interval_seconds}s "
                f"nodes={len(clusters)} edges={len(edges)}",
                flush=True,
            )
            time.sleep(config.poll_interval_seconds)
    finally:
        driver.close()


if __name__ == "__main__":
    run()

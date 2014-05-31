Disk access (even if mmapped) is slow primarily based on the number of disk seeks required to access data.

While mongo mmaps data into RAM that only provides a speedup if your data is already in RAM, if it's on disk it doesn't help. When you try to scan say 10k records in a mongo query, it must perform disk seeks on both the index and the data extents to complete a query. This creates a significant cold start problem.

mongosort attempts to sort ondisk data ordered by the primary key `_id` so that when using custom `_id` values and querying based on that sort order, it takes as few seeks as possible to map data in from disk.


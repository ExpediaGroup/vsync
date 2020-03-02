---
id: goodparts
title: Good Parts
sidebar_label: Good Parts
---

These are good parts of vsync, probably we should not change accidentally in future

* Don't change the destination metadata based on destination secrets, it is currently and it should be on origin metadata, because the destination updated time and will be different always. When we compare in next sync cycle the info will be different and we will forever be syncing

* Destination is halting if syncmap is not present in destination vsync/ in consul, which will need manual restart to re initialize the vsync path

* For transformer regex, use https://regex101.com/. Example: https://regex101.com/r/yelNjd/1
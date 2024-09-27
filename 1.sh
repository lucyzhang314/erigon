#!/bin/bash
echo $1

go run ./cmd/erigon snapshots uncompress /erigon-data/snapshots_v0/$1 | DictReducerSoftLimit=1000000 MinPatternLen=5 MaxPatternLen=128 SamplingFactor=1 MaxDictPatterns=65536 go run ./cmd/erigon snapshots compress --datadir=/erigon-data/erigon3/ /erigon-data/snapshots/$1

echo "$1 done"

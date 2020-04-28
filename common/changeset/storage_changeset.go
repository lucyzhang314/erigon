package changeset

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ledgerwatch/turbo-geth/common"
	"sort"
)

const (
	DefaultIncarnation                  = uint64(1)
	storageEnodingLengthOfUniqueElemens = 2
)

var ErrNotFound = errors.New("not found")

func NewStorageChangeSet() *ChangeSet {
	return &ChangeSet{
		Changes: make([]Change, 0),
		keyLen:  2*common.HashLength + common.IncarnationLength,
	}
}

/**
numOfElements uint32
numOfUniqueContracts uint16
[]{
	addrhash common.Hash
	numOfStorageKeysToSkip uint16
}
keys     []common.Hash numOfElements
 lenToValue []uint16,
numOfUint8Values uint16
numOfUint16Values uint16
numOfUint32Values uint16
[len(val0), len(val0)+len(val1), ..., len(val0)+len(val1)+...+len(val_{numOfUint8Values-1})] []uint8
[len(valnumOfUint8Values), len(val0)+len(val1), ..., len(val0)+len(val1)+...+len(val_{numOfUint16Values-1})] []uint16
[len(valnumOfUint16Values), len(val0)+len(val1), ..., len(val0)+len(val1)+...+len(val_{numOfUint32Values-1})] []uint32
[elementNum:incarnation] -  optional [uint32:uint64...]

*/

func EncodeStorage(s *ChangeSet) ([]byte, error) {
	sort.Sort(s)
	var err error
	buf := new(bytes.Buffer)
	uint16Arr := make([]byte, 2)
	uint32Arr := make([]byte, 4)
	numOfElements := s.Len()

	keys := make([]contractKeys, 0, numOfElements)
	valLengthes := make([]byte, 0, numOfElements)
	var (
		currentContract contractKeys
		numOfUint8      uint16
		numOfUint16     uint16
		numOfUint32     uint16
		lengthOfValues  uint32
	)
	var nonDefaultIncarnationCounter uint16
	notDefaultIncarnationsBytes := make([]byte, 2)
	b := make([]byte, 10)

	currentKey := -1
	for i, change := range s.Changes {
		addrHash := change.Key[0:common.HashLength]
		incarnation := binary.BigEndian.Uint64(change.Key[common.HashLength : common.HashLength+common.IncarnationLength])
		keyHash := change.Key[common.HashLength+common.IncarnationLength : 2*common.HashLength+common.IncarnationLength]
		//found new contract address
		if i == 0 || !bytes.Equal(currentContract.AddrHash, addrHash) || currentContract.Incarnation != incarnation {
			currentKey++
			currentContract.AddrHash = addrHash
			currentContract.Incarnation = incarnation
			//add to incarnations part
			if incarnation != DefaultIncarnation {
				binary.BigEndian.PutUint16(b[0:2], uint16(currentKey))
				binary.BigEndian.PutUint64(b[2:10], incarnation)
				notDefaultIncarnationsBytes = append(notDefaultIncarnationsBytes, b...)
				nonDefaultIncarnationCounter++
			}
			currentContract.Keys = [][]byte{keyHash}
			currentContract.Vals = [][]byte{change.Value}
			keys = append(keys, currentContract)
		} else {
			//add key and value
			currentContract.Keys = append(currentContract.Keys, keyHash)
			currentContract.Vals = append(currentContract.Vals, change.Value)
		}

		//calculate lengthes of values
		lengthOfValues += uint32(len(change.Value))
		switch {
		case lengthOfValues <= 255:
			valLengthes = append(valLengthes, uint8(lengthOfValues))
			numOfUint8++

		case lengthOfValues <= 65535:
			binary.BigEndian.PutUint16(uint16Arr, uint16(lengthOfValues))
			valLengthes = append(valLengthes, uint16Arr...)
			numOfUint16++

		default:
			binary.BigEndian.PutUint32(uint32Arr, lengthOfValues)
			valLengthes = append(valLengthes, uint32Arr...)
			numOfUint32++
		}

		//save to array
		keys[currentKey] = currentContract
	}

	// save numOfUniqueContracts
	binary.BigEndian.PutUint16(uint16Arr, uint16(len(keys)))
	if _, err = buf.Write(uint16Arr); err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, errors.New("empty prepared data")
	}

	var endNumOfKeys int

	for i := 0; i < len(keys); i++ {
		if _, err = buf.Write(keys[i].AddrHash); err != nil {
			return nil, err
		}
		endNumOfKeys += len(keys[i].Keys)

		//end of keys
		binary.BigEndian.PutUint16(uint16Arr, uint16(endNumOfKeys))
		if _, err = buf.Write(uint16Arr); err != nil {
			return nil, err
		}
	}

	if endNumOfKeys != numOfElements {
		return nil, fmt.Errorf("incorrect number of elements must:%v current:%v", numOfElements, endNumOfKeys)
	}

	binary.BigEndian.PutUint16(notDefaultIncarnationsBytes, nonDefaultIncarnationCounter)
	if _, err = buf.Write(notDefaultIncarnationsBytes); err != nil {
		return nil, err
	}

	for _, group := range keys {
		for _, v := range group.Keys {
			if _, err = buf.Write(v); err != nil {
				return nil, err
			}
		}
	}

	binary.BigEndian.PutUint16(uint16Arr, numOfUint8)
	if _, err = buf.Write(uint16Arr); err != nil {
		return nil, err
	}

	binary.BigEndian.PutUint16(uint16Arr, numOfUint16)
	if _, err = buf.Write(uint16Arr); err != nil {
		return nil, err
	}

	binary.BigEndian.PutUint16(uint16Arr, numOfUint32)
	if _, err = buf.Write(uint16Arr); err != nil {
		return nil, err
	}

	binary.BigEndian.PutUint32(uint32Arr, lengthOfValues)
	if _, err = buf.Write(uint32Arr); err != nil {
		return nil, err
	}

	if _, err = buf.Write(valLengthes); err != nil {
		return nil, err
	}

	for _, v := range keys {
		for _, val := range v.Vals {
			if _, err = buf.Write(val); err != nil {
				return nil, err
			}
		}
	}
	return buf.Bytes(), nil
}

func DecodeStorage(b []byte) (*ChangeSet, error) {
	numOfUniqueElements := int(binary.BigEndian.Uint16(b[4:6]))
	if numOfUniqueElements == 0 {
		return &ChangeSet{
			Changes: make([]Change, 0),
			keyLen:  72,
		}, nil
	}
	keys := make([]contractKeys, numOfUniqueElements)
	numOfSkipKeys := make([]int, numOfUniqueElements+1)
	for i := uint32(0); i < uint32(numOfUniqueElements); i++ {
		numOfSkipKeys[i+1] = int(binary.BigEndian.Uint16(b[2+i*(common.HashLength)+(i-1)*2 : 2+i*(common.HashLength)+(i)*2]))
		start := 2 + i*common.HashLength + i*2
		keys[i].AddrHash = b[start : start+common.HashLength]
		keys[i].Incarnation = DefaultIncarnation
	}
	numOfElements := numOfSkipKeys[numOfUniqueElements]
	incarnatonsInfo := 2 + numOfUniqueElements*(common.HashLength+2)
	numOfNotDefaultIncarnations := int(binary.BigEndian.Uint16(b[incarnatonsInfo : incarnatonsInfo+2]))

	incarnationsStart := incarnatonsInfo + 2
	if numOfNotDefaultIncarnations > 0 {
		for i := 0; i < numOfNotDefaultIncarnations; i++ {
			id := binary.BigEndian.Uint16(b[incarnationsStart+i*10 : incarnationsStart+i*10+2])
			keys[id].Incarnation = binary.BigEndian.Uint64(b[incarnationsStart+i*10+2 : incarnationsStart+i*10+10])
		}
	}

	keysStart := incarnationsStart + numOfNotDefaultIncarnations*10

	for i := 0; i < numOfUniqueElements; i++ {
		keys[i].Keys = make([][]byte, 0, numOfSkipKeys[i+1]-numOfSkipKeys[i])
		for j := numOfSkipKeys[i]; j < numOfSkipKeys[i+1]; j++ {
			keys[i].Keys = append(keys[i].Keys, b[keysStart+j*common.HashLength:keysStart+(j+1)*common.HashLength])
		}
	}

	valsInfoStart := keysStart + numOfElements*common.HashLength
	cs := NewStorageChangeSet()
	cs.Changes = make([]Change, numOfElements)
	id := 0
	for _, v := range keys {
		for i := range v.Keys {
			k := make([]byte, common.HashLength*2+common.IncarnationLength)
			copy(k[:common.HashLength], v.AddrHash)
			binary.BigEndian.PutUint64(k[common.HashLength:common.HashLength+common.IncarnationLength], v.Incarnation)
			copy(k[common.HashLength+common.IncarnationLength:common.HashLength*2+common.IncarnationLength], v.Keys[i])
			val, innerErr := FindValue(b[valsInfoStart:], id)
			if innerErr != nil {
				return nil, innerErr
			}
			cs.Changes[id] = Change{
				Key:   k,
				Value: val,
			}

			id++
		}
	}

	return cs, nil
}

func FindValue(b []byte, i int) ([]byte, error) {
	numOfUint8 := int(binary.BigEndian.Uint16(b[0:2]))
	numOfUint16 := int(binary.BigEndian.Uint16(b[2:4]))
	numOfUint32 := int(binary.BigEndian.Uint16(b[4:6]))
	lenOfValsStartPointer := 10
	valsPointer := lenOfValsStartPointer + numOfUint8 + numOfUint16*2 + numOfUint32*4
	var (
		lenOfValStart int
		lenOfValEnd   int
	)

	switch {
	case i < numOfUint8:
		lenOfValEnd = int(b[lenOfValsStartPointer+i])
		if i > 0 {
			lenOfValStart = int(b[lenOfValsStartPointer+i-1])
		}
	case i < numOfUint8+numOfUint16:
		one := (i-numOfUint8)*2 + numOfUint8
		lenOfValEnd = int(binary.BigEndian.Uint16(b[lenOfValsStartPointer+one : lenOfValsStartPointer+one+2]))
		if i-1 < numOfUint8 {
			lenOfValStart = int(b[lenOfValsStartPointer+i-1])
		} else {
			one = (i-1)*2 - numOfUint8
			lenOfValStart = int(binary.BigEndian.Uint16(b[lenOfValsStartPointer+one : lenOfValsStartPointer+one+2]))
		}
	case i < numOfUint8+numOfUint16+numOfUint32:
		one := lenOfValsStartPointer + numOfUint8 + numOfUint16*2 + (i-numOfUint8-numOfUint16)*4
		lenOfValEnd = int(binary.BigEndian.Uint32(b[one : one+4]))
		if i-1 < numOfUint8+numOfUint16 {
			one = lenOfValsStartPointer + (i-1)*2 - numOfUint8
			lenOfValStart = int(binary.BigEndian.Uint16(b[one : one+2]))
		} else {
			one = lenOfValsStartPointer + numOfUint8 + numOfUint16*2 + (i-1-numOfUint8-numOfUint16)*4
			lenOfValStart = int(binary.BigEndian.Uint32(b[one : one+4]))
		}
	default:
		return nil, errors.New("find value error")
	}
	return common.CopyBytes(b[valsPointer+lenOfValStart : valsPointer+lenOfValEnd]), nil
}

type StorageChangeSetBytes []byte

func (b StorageChangeSetBytes) Walk(f func(k, v []byte) error) error {
	if len(b) == 0 {
		return nil
	}

	if len(b) < 4 {
		return fmt.Errorf("decode: input too short (%d bytes)", len(b))
	}

	numOfUniqueElements := int(binary.BigEndian.Uint16(b))
	if numOfUniqueElements == 0 {
		return nil
	}
	incarnatonsInfo := 2 + numOfUniqueElements*(common.HashLength+2)
	numOfNotDefaultIncarnations := int(binary.BigEndian.Uint16(b[incarnatonsInfo:]))
	incarnatonsStart := incarnatonsInfo + 2

	notDefaultIncarnations := make(map[uint16]uint64)
	if numOfNotDefaultIncarnations > 0 {
		for i := 0; i < numOfNotDefaultIncarnations; i++ {
			notDefaultIncarnations[binary.BigEndian.Uint16(b[incarnatonsStart+i*10:])] = binary.BigEndian.Uint64(b[incarnatonsStart+i*10+2:])
		}
	}

	keysStart := incarnatonsStart + numOfNotDefaultIncarnations*10
	numOfElements := int(binary.BigEndian.Uint16(b[incarnatonsInfo-2:]))
	valsInfoStart := keysStart + numOfElements*common.HashLength

	var addressHashID uint16
	var id int
	for i := 0; i < numOfUniqueElements; i++ {
		var (
			startKeys int
			endKeys   int
		)

		if i > 0 {
			startKeys = int(binary.BigEndian.Uint16(b[2+i*(common.HashLength)+(i-1)*2:]))
		}
		endKeys = int(binary.BigEndian.Uint16(b[2+(i+1)*(common.HashLength)+i*2:]))
		addrHash := b[2+i*(common.HashLength)+i*2:]
		incarnation := DefaultIncarnation
		if inc, ok := notDefaultIncarnations[addressHashID]; ok {
			incarnation = inc
		}

		for j := startKeys; j < endKeys; j++ {
			k := make([]byte, 72)
			copy(k[:common.HashLength], addrHash[:common.HashLength])
			copy(k[common.HashLength+common.IncarnationLength:2*common.HashLength+common.IncarnationLength], b[keysStart+j*common.HashLength:])
			binary.BigEndian.PutUint64(k[common.HashLength:], incarnation)
			val, innerErr := FindValue(b[valsInfoStart:], id)
			if innerErr != nil {
				return innerErr
			}
			err := f(k, val)
			if err != nil {
				return err
			}
			id++
		}
		addressHashID++
	}

	return nil
}

func (b StorageChangeSetBytes) Find(k []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, nil
	}
	if len(b) < 4 {
		return nil, fmt.Errorf("decode: input too short (%d bytes)", len(b))
	}

	numOfUniqueElements := int(binary.BigEndian.Uint16(b))
	if numOfUniqueElements == 0 {
		return nil, nil
	}
	numOfElements := int(binary.BigEndian.Uint16(b[2+(numOfUniqueElements-1)*(common.HashLength+2)+common.HashLength:]))
	incarnatonsInfo := 2 + numOfUniqueElements*(common.HashLength+2)
	numOfNotDefaultIncarnations := int(binary.BigEndian.Uint16(b[incarnatonsInfo:]))
	incarnatonsStart := incarnatonsInfo + 2
	keysStart := incarnatonsStart + numOfNotDefaultIncarnations*10
	valsInfoStart := keysStart + numOfElements*common.HashLength

	addHashID := sort.Search(numOfUniqueElements, func(i int) bool {
		addrHash := b[2+i*(2+common.HashLength) : 2+i*(2+common.HashLength)+common.HashLength]
		cmp := bytes.Compare(addrHash, k[0:common.HashLength])
		return cmp >= 0
	})

	if addHashID == numOfUniqueElements {
		return nil, ErrNotFound
	}

	from := 0
	if addHashID > 0 {
		from = int(binary.BigEndian.Uint16(b[2+addHashID*common.HashLength+addHashID*2-2:]))
	}
	to := int(binary.BigEndian.Uint16(b[2+(addHashID+1)*common.HashLength+addHashID*2:]))

	keyIndex := sort.Search(to-from, func(i int) bool {
		index := from + i
		key := b[keysStart+common.HashLength*index : keysStart+common.HashLength*index+common.HashLength]
		cmp := bytes.Compare(key, k[common.HashLength+common.IncarnationLength:2*common.HashLength+common.IncarnationLength])
		return cmp >= 0
	})
	index := from + keyIndex
	if index == to {
		return nil, ErrNotFound
	}
	return FindValue(b[valsInfoStart:], index)
}

func (b StorageChangeSetBytes) FindWithoutIncarnation(addrHashToFind []byte, keyHashToFind []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, nil
	}
	if len(b) < 4 {
		return nil, fmt.Errorf("decode: input too short (%d bytes)", len(b))
	}

	numOfUniqueElements := int(binary.BigEndian.Uint16(b))
	if numOfUniqueElements == 0 {
		return nil, nil
	}
	numOfElements := int(binary.BigEndian.Uint16(b[2+(numOfUniqueElements-1)*(common.HashLength+2)+common.HashLength:]))
	incarnatonsInfo := 2 + numOfUniqueElements*(common.HashLength+2)
	numOfNotDefaultIncarnations := int(binary.BigEndian.Uint16(b[incarnatonsInfo:]))
	incarnatonsStart := incarnatonsInfo + 2
	keysStart := incarnatonsStart + numOfNotDefaultIncarnations*10
	valsInfoStart := keysStart + numOfElements*common.HashLength

	addHashID := sort.Search(int(numOfUniqueElements), func(i int) bool {
		addrHash := b[2+i*(common.HashLength)+i*2 : 2+(i+1)*(common.HashLength)+i*2]
		cmp := bytes.Compare(addrHash, addrHashToFind)
		return cmp >= 0

	})

	if addHashID >= numOfUniqueElements {
		return nil, ErrNotFound
	}
	from := 0
	if addHashID > 0 {
		from = int(binary.BigEndian.Uint16(b[2+addHashID*common.HashLength+addHashID*2-2:]))
	}
	to := int(binary.BigEndian.Uint16(b[2+(addHashID+1)*common.HashLength+addHashID*2:]))
	keyIndex := sort.Search(to-from, func(i int) bool {
		index := from + i
		key := b[keysStart+common.HashLength*index : keysStart+common.HashLength*index+common.HashLength]
		cmp := bytes.Compare(key, keyHashToFind)
		return cmp >= 0
	})
	index := from + keyIndex
	if index == to-from {
		return nil, ErrNotFound
	}

	return FindValue(b[valsInfoStart:], index)
}

type contractKeys struct {
	AddrHash    []byte
	Incarnation uint64
	Keys        [][]byte
	Vals        [][]byte
}

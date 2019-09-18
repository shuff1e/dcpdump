+ 由于抓包时，只截取了1024字节，因此解析时，只解析到key(1024字节一般足够了)，可能无法解析整个数据包，所以需要修改一下gomemcached的代码

+ modify *mc_req.go*  

```
// Receive will fill this MCRequest with the data from a reader.
func (req *MCRequest) Receive(r io.Reader, hdrBytes []byte) (int, error) {
	if len(hdrBytes) < HDR_LEN {
		hdrBytes = []byte{
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0}
	}
	n, err := io.ReadFull(r, hdrBytes)
	if err != nil {
		return n, err
	}

	if hdrBytes[0] != RES_MAGIC && hdrBytes[0] != REQ_MAGIC {
		return n, fmt.Errorf("bad magic: 0x%02x", hdrBytes[0])
	}

	klen := int(binary.BigEndian.Uint16(hdrBytes[2:]))
	elen := int(hdrBytes[4])
	// Data type at 5
	req.DataType = uint8(hdrBytes[5])

	req.Opcode = CommandCode(hdrBytes[1])
	// Vbucket at 6:7
	req.VBucket = binary.BigEndian.Uint16(hdrBytes[6:])
	//totalBodyLen := int(binary.BigEndian.Uint32(hdrBytes[8:]))

	req.Opaque = binary.BigEndian.Uint32(hdrBytes[12:])
	req.Cas = binary.BigEndian.Uint64(hdrBytes[16:])

	if elen + klen > 0 {
		buf := make([]byte, elen + klen)
		m, err := io.ReadFull(r, buf)
		n += m
		if err == nil {
			if req.Opcode >= TAP_MUTATION &&
				req.Opcode <= TAP_CHECKPOINT_END &&
				len(buf) > 1 {
				// In these commands there is "engine private"
				// data at the end of the extras.  The first 2
				// bytes of extra data give its length.
				elen += int(binary.BigEndian.Uint16(buf))
			}

			req.Extras = buf[0:elen]
			req.Key = buf[elen : klen+elen]

			// get the length of extended metadata
			//extMetaLen := 0
			//if elen > 29 {
			//	extMetaLen = int(binary.BigEndian.Uint16(req.Extras[28:30]))
			//}

			//bodyLen := totalBodyLen - klen - elen - extMetaLen
			//if bodyLen > MaxBodyLen {
			//	return n, fmt.Errorf("%d is too big (max %d)",
			//		bodyLen, MaxBodyLen)
			//}
			//
			//req.Body = buf[klen+elen : klen+elen+bodyLen]
			//req.ExtMeta = buf[klen+elen+bodyLen:]
		}
	}
	return n, err
}
```

+ modify *mc_res.go*  

```
// Receive will fill this MCResponse with the data from this reader.
func (res *MCResponse) Receive(r io.Reader, hdrBytes []byte) (n int, err error) {
	if len(hdrBytes) < HDR_LEN {
		hdrBytes = []byte{
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0}
	}
	n, err = io.ReadFull(r, hdrBytes)
	if err != nil {
		return n, err
	}

	if hdrBytes[0] != RES_MAGIC && hdrBytes[0] != REQ_MAGIC {
		return n, fmt.Errorf("bad magic: 0x%02x", hdrBytes[0])
	}

	klen := int(binary.BigEndian.Uint16(hdrBytes[2:4]))
	elen := int(hdrBytes[4])

	res.Opcode = CommandCode(hdrBytes[1])
	res.DataType = uint8(hdrBytes[5])
	res.Status = Status(binary.BigEndian.Uint16(hdrBytes[6:8]))
	res.Opaque = binary.BigEndian.Uint32(hdrBytes[12:16])
	res.Cas = binary.BigEndian.Uint64(hdrBytes[16:24])

	bodyLen := int(binary.BigEndian.Uint32(hdrBytes[8:12])) - (klen + elen)

	//defer function to debug the panic seen with MB-15557
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf(`Panic in Receive. Response %v \n
                        key len %v extra len %v bodylen %v`, res, klen, elen, bodyLen)
		}
	}()

	buf := make([]byte, klen+elen)
	m, err := io.ReadFull(r, buf)
	if err == nil {
		res.Extras = buf[0:elen]
		res.Key = buf[elen : klen+elen]
		//res.Body = buf[klen+elen:]
	}

	return n + m, err
}
```
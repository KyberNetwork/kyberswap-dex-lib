package poolmanager

// ShareCachedPoolManager share pools by caching IPool with ttl
// Note: it is not currently used
//type ShareCachedPoolManager struct {
//	cache        *ttlcache.Cache[string, pool.IPool]
//	poolEntities cmap.ConcurrentMap
//	config       Config
//
//	poolFactory    IPoolFactory
//	poolRepository IPoolRepository
//}
//
//func NewShareCachedPoolManager(
//	poolRepository IPoolRepository,
//	poolFactory IPoolFactory,
//	config Config,
//) *ShareCachedPoolManager {
//	var (
//		c *ttlcache.Cache[string, pool.IPool]
//	)
//	if config.PoolRenewalInterval > 0 {
//		c = ttlcache.New[string, pool.IPool](
//			ttlcache.WithTTL[string, pool.IPool](config.PoolRenewalInterval),
//			ttlcache.WithCapacity[string, pool.IPool](config.Capacity),
//		)
//		go c.Start()
//	}
//
//	return &ShareCachedPoolManager{
//		cache:          c,
//		poolRepository: poolRepository,
//		poolFactory:    poolFactory,
//		config:         config,
//		poolEntities:   cmap.New(),
//	}
//}
//
//// updateCache should be called in a separated thread once there is new pools
//func (p *ShareCachedPoolManager) updateCache(newPoolStruct map[string]*entity.Pool, newPoolInterface map[string]pool.IPool) {
//	for k := range newPoolInterface {
//		p.cache.Set(k, newPoolInterface[k], p.config.PoolRenewalInterval)
//	}
//	for k := range newPoolStruct {
//		p.poolEntities.Set(k, newPoolStruct[k])
//	}
//}
//
//// GetPools return a list  of physical entity and a map of IPool interface from them
//// WARNING: these two lists are not in the same length. FindRoute request should use only the IPool interface
//func (p *ShareCachedPoolManager) GetPools(ctx context.Context, ids sets.String) (map[string]*entity.Pool, map[string]pool.IPool, error) {
//	var (
//		item                    *ttlcache.Item[string, pool.IPool]
//		addressToReadFromDB     []string
//		results                 = make(map[string]pool.IPool, len(ids))
//		newEntityPoolsByAddress = make(map[string]*entity.Pool, len(ids))
//	)
//
//	for _, key := range ids.List() {
//		if item = p.cache.Get(key); item == nil {
//			addressToReadFromDB = append(addressToReadFromDB, key)
//		} else {
//			results[key] = item.Value()
//			if results[key] == nil {
//				panic("motherfucker")
//			}
//		}
//	}
//
//	newEntityPools, err := p.poolRepository.FindByAddresses(ctx, addressToReadFromDB)
//	if err != nil {
//		return nil, nil, err
//	}
//	for i := range newEntityPools {
//		newEntityPoolsByAddress[newEntityPools[i].Address] = newEntityPools[i]
//	}
//	newIPools := p.poolFactory.NewPoolByAddress(ctx, newEntityPools)
//
//	//fmt.Println("len of newEntityPools %d newIPools %D", len(newEntityPools), len(newIPools))
//
//	for key := range newIPools {
//		if newIPools[key] == nil {
//			panic("what the fuck is going on")
//		}
//		results[key] = newIPools[key]
//	}
//
//	return newEntityPoolsByAddress, results, nil
//}
//
//// TODO: fix this interface
//func (p *ShareCachedPoolManager) GetPoolsAndItsBase(ctx context.Context, ids sets.String, dexSet sets.String) (map[string]pool.IPool, error) {
//	result := make(map[string]pool.IPool, len(ids))
//
//	newPools, poolInterfacesByAddress, err := p.GetPools(ctx, ids)
//
//	//fmt.Println("len of newPools %d poolInterfacesByAddress %D", len(newPools), len(poolInterfacesByAddress))
//	if err != nil {
//		return nil, err
//	}
//
//	//for each of these newPools, check if its basePool is also loaded
//	var basePoolsToGet = sets.NewString()
//	for key := range poolInterfacesByAddress {
//		poolStruct := newPools[key]
//
//		if poolStruct == nil {
//			//not a new pool, look into old pools data
//			data, avail := p.poolEntities.Get(key)
//			if !avail {
//				continue
//			}
//			poolStruct = data.(*entity.Pool)
//		}
//		if dexSet.Has(poolStruct.Address) && (poolStruct.HasReserves() || poolStruct.HasAmplifiedTvl()) {
//			if poolInterfacesByAddress[poolStruct.Address] == nil {
//				panic("fuck off")
//			}
//			result[poolStruct.Address] = poolInterfacesByAddress[poolStruct.Address]
//		}
//		if poolStruct.Type == constant.PoolTypes.CurveMeta {
//			unmarshalledPool, ok := poolInterfacesByAddress[poolStruct.Address].(*curveMeta.Pool)
//			if !ok {
//				continue
//			}
//			basePoolsToGet = basePoolsToGet.Insert(unmarshalledPool.BasePool.GetInfo().Address)
//		}
//	}
//
//	remainingPools := basePoolsToGet.Difference(ids)
//
//	// If there are more basePools to get, get them as well
//	if remainingPools.Len() > 0 {
//		newBasePools, basePoolsInterfacesByAddress, err := p.GetPools(ctx, remainingPools)
//		if err != nil {
//			return nil, err
//		}
//		//append the results
//		for key := range basePoolsInterfacesByAddress {
//			poolStruct := newBasePools[key]
//
//			if poolStruct == nil {
//				//not a new pool, look into old pools data
//				data, avail := p.poolEntities.Get(key)
//				if !avail {
//					continue
//				}
//				poolStruct = data.(*entity.Pool)
//			}
//			if dexSet.Has(poolStruct.Exchange) && (poolStruct.HasReserves() || poolStruct.HasAmplifiedTvl()) {
//				if basePoolsInterfacesByAddress[poolStruct.Address] == nil {
//					panic("dead shit")
//				}
//				result[poolStruct.Address] = basePoolsInterfacesByAddress[poolStruct.Address]
//			}
//		}
//		//merge the basePools and the pools got from previous steps
//		for i := range newBasePools {
//			newPools[newBasePools[i].Address] = newBasePools[i]
//		}
//		for k := range basePoolsInterfacesByAddress {
//			poolInterfacesByAddress[k] = basePoolsInterfacesByAddress[k]
//		}
//	}
//
//	p.updateCache(newPools, poolInterfacesByAddress)
//	return result, nil
//}

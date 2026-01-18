import { describe, it, expect } from 'vitest'
import { convertTrafficUnits, formatTraffic, formatTrafficFromBytes } from './utils'

describe('Traffic unit conversion', () => {
  describe('convertTrafficUnits', () => {
    it('should return MB for values < 1000 MB', () => {
      const result = convertTrafficUnits(999)
      expect(result).toEqual({ value: 999, unit: 'MB' })
    })

    it('should convert 1000 MB to 1.00 GB', () => {
      const result = convertTrafficUnits(1000)
      expect(result).toEqual({ value: 1, unit: 'GB' })
    })

    it('should convert 1500 GB to 1.50 TB', () => {
      const result = convertTrafficUnits(1500 * 1000) // 1500 GB in MB
      expect(result).toEqual({ value: 1.5, unit: 'TB' })
    })

    it('should convert 1000 TB to 1.00 PB', () => {
      const result = convertTrafficUnits(1000 * 1000 * 1000) // 1000 TB in MB
      expect(result).toEqual({ value: 1, unit: 'PB' })
    })

    it('should handle large PB values', () => {
      const result = convertTrafficUnits(2500 * 1000 * 1000 * 1000) // 2500 TB in MB
      expect(result).toEqual({ value: 2.5, unit: 'PB' })
    })
  })

  describe('formatTraffic', () => {
    it('should format 999 MB as "999.00 MB"', () => {
      expect(formatTraffic(999)).toBe('999.00 MB')
    })

    it('should format 1000 MB as "1.00 GB"', () => {
      expect(formatTraffic(1000)).toBe('1.00 GB')
    })

    it('should format 1500 GB as "1.50 TB"', () => {
      expect(formatTraffic(1500 * 1000)).toBe('1.50 TB')
    })

    it('should format 1000 TB as "1.00 PB"', () => {
      expect(formatTraffic(1000 * 1000 * 1000)).toBe('1.00 PB')
    })

    it('should format without unit when includeUnit is false', () => {
      expect(formatTraffic(1500 * 1000, false)).toBe('1.50')
    })
  })

  describe('formatTrafficFromBytes', () => {
    it('should convert bytes to appropriate traffic units', () => {
      // 1 GB = 1024 MB = 1024 * 1024 * 1024 bytes
      const bytes = 1024 * 1024 * 1024
      expect(formatTrafficFromBytes(bytes)).toBe('1.07 GB')
    })

    it('should handle small byte values', () => {
      const bytes = 1024 * 1024 // 1 MB
      expect(formatTrafficFromBytes(bytes)).toBe('1.05 MB')
    })
  })
})
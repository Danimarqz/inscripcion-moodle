export interface LogStatsRange {
  from?: string;
  to?: string;
  site?: string;
}

export interface DayBucket {
  date: string;
  hits: number;
}

export interface MonthBucket {
  month: string;
  hits: number;
}

export interface URLBucket {
  url: string;
  hits: number;
}

export interface IPBucket {
  ip: string;
  hits: number;
}

export interface StatusBucket {
  status: number;
  hits: number;
}

export interface MethodBucket {
  method: string;
  hits: number;
}

export interface UABucket {
  userAgent: string;
  hits: number;
}

export interface LogStats {
  range: LogStatsRange;
  totalHits: number;
  botHits: number;
  scannerHits: number;
  humanHits: number;
  uniqueIPs: number;
  byDay: DayBucket[];
  byMonth: MonthBucket[];
  topUrls: URLBucket[];
  topIps: IPBucket[];
  byStatus: StatusBucket[];
  sites: string[];
  topMethods: MethodBucket[];
  topUA: UABucket[];
  topBotUA: UABucket[];
  topScannerPath: URLBucket[];
}

export interface LogStatsQuery {
  from?: string;
  to?: string;
  site?: string;
  topN?: number;
}

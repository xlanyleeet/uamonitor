export interface Status { date: string; isOnline: boolean; onlineCount: number | null; maxOnline: number | null; version: string | null; }
export interface GraphPointData { isOnline: boolean; onlineCount: number; date: string; }
export interface GraphServer { id: number; data: GraphPointData[]; }
export interface Server {
  id: number; name: string; ip: string; port: string;
  icon: string | null; link: string | null; telegram: string | null;
  discord: string | null; youtube: string | null; description: string | null;
  uptime?: { days: number; percent: number | null; coverage: number };
  statuses: Status[];
}
export interface ParsedServer extends Server { graph: GraphServer; }

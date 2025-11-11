import { Injectable } from '@angular/core';
import { Observable, BehaviorSubject, of, interval, map, tap } from 'rxjs';
import { ScanRequest, ScanTask, ScanDetail } from '../models/scan.model';

@Injectable({ providedIn: 'root' })
export class ScanService {
  // Initialize details before tasks$, because loadInitial() uses this.details
  private details = new Map<string, ScanDetail>();
  private tasks$ = new BehaviorSubject<ScanTask[]>(this.loadInitial());

  createScan(request: ScanRequest): Observable<ScanTask> {
    const id = Math.random().toString(36).slice(2, 10) + Math.random().toString(36).slice(2, 10);
    const now = new Date().toISOString();
    const task: ScanDetail = {
      id,
      targets: request.targets,
      ports: request.ports,
      mode: request.mode,
      status: 'running',
      created_at: now,
      started_at: now,
      total_hosts: request.targets.length || 1,
      scanned_hosts: 0,
      total_ports: 100,
      open_ports: 0,
      results: []
    };
    this.details.set(id, task);
    this.tasks$.next([task, ...this.tasks$.value]);
    return of(task);
  }

  getScanById(id: string): Observable<ScanDetail> {
    const found = this.details.get(id) || this.tasks$.value.find(t => t.id === id);
    const detail: ScanDetail = found as ScanDetail;
    return of(detail);
  }

  getAllScans(): Observable<ScanTask[]> {
    return this.tasks$.asObservable();
  }

  pollScanStatus(id: string, intervalMs: number = 1500): Observable<ScanDetail> {
    return interval(intervalMs).pipe(
      map(() => {
        const d = this.details.get(id);
        if (!d) return undefined as unknown as ScanDetail;
        if (d.status === 'running') {
          d.scanned_hosts = Math.min((d.scanned_hosts || 0) + 1, d.total_hosts || 1);
          if ((d.results?.length || 0) < 3) {
            d.results = [...(d.results || []), { host: d.targets[0] || '127.0.0.1', port: 80 + (d.results?.length || 0), state: 'open', service: 'http' }];
            d.open_ports = (d.results?.length || 0);
          }
          if ((d.scanned_hosts || 0) >= (d.total_hosts || 1)) {
            d.status = 'completed';
            d.completed_at = new Date().toISOString();
          }
          this.details.set(id, { ...d });
          this.tasks$.next(this.tasks$.value.map(t => (t.id === id ? { ...t, ...d } : t)));
        }
        return this.details.get(id)!;
      })
    );
  }

  private loadInitial(): ScanTask[] {
    const now = new Date();
    const mk = (i: number): ScanTask => ({
      id: (i + 1).toString(16).padStart(8, '0') + 'deadbeef',
      targets: ['scanme.nmap.org'],
      ports: '22,80,443',
      mode: (i % 3 === 0 ? 'CONNECT' : i % 3 === 1 ? 'SYN' : 'UDP') as any,
      status: (i % 4 === 0 ? 'running' : i % 4 === 1 ? 'completed' : i % 4 === 2 ? 'failed' : 'pending'),
      created_at: new Date(now.getTime() - i * 3600_000).toISOString(),
      total_hosts: 3,
      scanned_hosts: i % 4 === 0 ? 1 : 3,
      open_ports: i % 2
    });
    const arr = Array.from({ length: 6 }, (_, i) => mk(i));
    for (const t of arr) {
      this.details.set(t.id, { ...t, results: [] });
    }
    return arr;
  }
}

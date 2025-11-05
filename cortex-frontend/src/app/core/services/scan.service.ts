import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable, timer } from 'rxjs';
import { map, switchMap, takeWhile, tap } from 'rxjs/operators';
import { environment } from '../../../environments/environment';

export type ScanMode = 'connect' | 'syn' | 'udp';

export interface CreateScanRequest {
  hosts: string[];
  ports: string; // e.g., "22,80,443,1000-1100"
  mode: ScanMode;
}

export interface ScanAcceptedResponse { id: string; status: 'pending'; }

export interface ScanResult { host: string; port: number; state: string; service?: string; }

export interface ScanTask {
  id: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  hosts: string[];
  ports: string;
  mode: ScanMode;
  created_at?: string;
  completed_at?: string | null;
  error?: string;
  results?: ScanResult[];
}

const STORAGE_KEY = 'cortex_tasks';

@Injectable({ providedIn: 'root' })
export class ScanService {
  private tasksSubject = new BehaviorSubject<ScanTask[]>(this.loadTasks());
  tasks$ = this.tasksSubject.asObservable();

  constructor(private http: HttpClient) {}

  createScan(req: CreateScanRequest): Observable<ScanAcceptedResponse> {
    return this.http.post<ScanAcceptedResponse>(`${environment.apiBaseUrl}/api/v1/scans`, req).pipe(
      tap((resp) => {
        const newTask: ScanTask = { id: resp.id, status: resp.status, hosts: req.hosts, ports: req.ports, mode: req.mode };
        this.upsertTask(newTask);
      })
    );
  }

  getScan(id: string): Observable<ScanTask> {
    return this.http.get<ScanTask>(`${environment.apiBaseUrl}/api/v1/scans/${id}`).pipe(
      tap((task) => this.upsertTask(task))
    );
  }

  pollScan(id: string, intervalMs = 3000): Observable<ScanTask> {
    return timer(0, intervalMs).pipe(
      switchMap(() => this.getScan(id)),
      takeWhile((t) => t.status === 'pending' || t.status === 'running', true)
    );
  }

  private upsertTask(task: ScanTask) {
    const current = this.tasksSubject.getValue();
    const idx = current.findIndex((t) => t.id === task.id);
    const updated = idx >= 0 ? [...current.slice(0, idx), { ...current[idx], ...task }, ...current.slice(idx + 1)] : [task, ...current];
    this.tasksSubject.next(updated);
    this.saveTasks(updated);
  }

  private loadTasks(): ScanTask[] {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      return raw ? JSON.parse(raw) : [];
    } catch {
      return [];
    }
  }

  private saveTasks(tasks: ScanTask[]) {
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(tasks.slice(0, 200))); } catch {}
  }
}

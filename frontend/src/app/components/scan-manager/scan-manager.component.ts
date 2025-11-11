import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ScanService } from '../../services/scan.service';
import { ScanTask, ScanRequest, ScanMode } from '../../models/scan.model';

interface ModeOption {
  mode: ScanMode;
  icon: string;
  title: string;
  description: string;
}

@Component({
  selector: 'app-scan-manager-skin',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './scan-manager.component.html',
  styleUrls: ['./scan-manager.component.scss']
})
export class ScanManagerComponent implements OnInit {
  modeOptions: ModeOption[] = [
    { mode: 'CONNECT', icon: '🔗', title: 'CONNECT SCAN', description: 'Fast, reliable scan with full handshake. Best for banner grabbing.' },
    { mode: 'SYN', icon: '⚡', title: 'SYN SCAN', description: 'Stealthy half-open scan. Lower detection rate, requires privileges.' },
    { mode: 'UDP', icon: '📡', title: 'UDP SCAN', description: 'Scans UDP services. Slower but discovers DNS, SNMP, etc.' }
  ];

  scanForm = { targets: '', ports: '1-1000', mode: 'CONNECT' as ScanMode, timeout: 5, max_concurrent_targets: 10 };

  allScans: ScanTask[] = [];
  filteredScans: ScanTask[] = [];
  statusFilter: 'all' | 'pending' | 'running' | 'completed' | 'failed' = 'all';
  loading = false;
  error = '';
  recentTargets: string[] = [];

  constructor(private scanService: ScanService, private router: Router) {}

  ngOnInit() { this.loadScans(); }

  loadScans() {
    this.scanService.getAllScans().subscribe({
      next: scans => {
        this.allScans = scans.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
        this.applyFilter();
        const bag = new Set<string>();
        for (const s of this.allScans) for (const t of (s.targets || [])) bag.add(t);
        this.recentTargets = Array.from(bag).slice(0, 6);
      },
      error: () => {}
    });
  }

  applyFilter() { this.filteredScans = this.statusFilter === 'all' ? this.allScans : this.allScans.filter(s => s.status === this.statusFilter); }
  selectMode(mode: ScanMode) { this.scanForm.mode = mode; }

  onSubmit() {
    if (!this.scanForm.targets.trim()) { this.error = 'Please enter at least one target'; return; }
    if (!this.scanForm.ports.trim()) { this.error = 'Please enter port range or list'; return; }
    this.loading = true; this.error = '';
    const request: ScanRequest = { targets: this.scanForm.targets.split(',').map(t => t.trim()), ports: this.scanForm.ports, mode: this.scanForm.mode, timeout: this.scanForm.timeout, max_concurrent_targets: this.scanForm.max_concurrent_targets };
    this.scanService.createScan(request).subscribe({ next: (task) => { this.router.navigate(['/scan-result', task.id]); }, error: () => { this.error = 'Failed to create scan. Please check your API key in Settings.'; this.loading = false; } });
  }

  viewScan(id: string) { this.router.navigate(['/scan-result', id]); }
  getStatusClass(status: string): string { return `status-${status}`; }
  formatDate(date: string): string { return new Date(date).toLocaleString(); }

  // Utilities card helpers
  applyPreset(ports: string) { this.scanForm.ports = ports; }
  addTarget(t: string) {
    const parts = this.scanForm.targets ? this.scanForm.targets.split(',').map(x => x.trim()).filter(Boolean) : [];
    if (!parts.includes(t)) parts.push(t);
    this.scanForm.targets = parts.join(', ');
  }
  get targetCount(): number { return this.scanForm.targets ? this.scanForm.targets.split(',').map(x => x.trim()).filter(Boolean).length : 0; }
  get estimatedPortCount(): number {
    const p = this.scanForm.ports || '';
    return p.split(',').map(x => x.trim()).filter(Boolean).reduce((sum, token) => {
      const m = token.match(/^(\d+)-(\d+)$/);
      if (m) { const a = parseInt(m[1],10), b = parseInt(m[2],10); if (!isNaN(a)&&!isNaN(b)&&b>=a) return sum + (b - a + 1); }
      const v = parseInt(token,10); return sum + (isNaN(v) ? 0 : 1);
    }, 0);
  }
}

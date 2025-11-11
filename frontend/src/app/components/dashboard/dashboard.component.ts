import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ScanService } from '../../services/scan.service';
import { ScanTask, ScanRequest } from '../../models/scan.model';

@Component({
  selector: 'app-dashboard-skin',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss']
})
export class DashboardComponent implements OnInit {
  quickScan = { targets: '', ports: '80,443,22,3389' };
  activeScans: ScanTask[] = [];
  loading = false;
  error = '';

  constructor(private scanService: ScanService, private router: Router) {}

  ngOnInit() { this.loadActiveScans(); }

  loadActiveScans() {
    this.scanService.getAllScans().subscribe({
      next: scans => { this.activeScans = scans.filter(s => s.status === 'running').slice(0, 5); },
      error: () => {}
    });
  }

  onQuickScan() {
    if (!this.quickScan.targets.trim()) { this.error = 'Please enter at least one target'; return; }
    this.loading = true; this.error = '';
    const request: ScanRequest = { targets: this.quickScan.targets.split(',').map(t => t.trim()), ports: this.quickScan.ports, mode: 'CONNECT' };
    this.scanService.createScan(request).subscribe({
      next: (task) => { this.router.navigate(['/scan-result', task.id]); },
      error: () => { this.error = 'Failed to create scan. Please check your API key in Settings.'; this.loading = false; }
    });
  }

  viewScan(id: string) { this.router.navigate(['/scan-result', id]); }
  getStatusClass(status: string): string { return `status-${status}`; }
}


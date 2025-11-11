import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { ScanService } from '../../services/scan.service';
import { ScanDetail } from '../../models/scan.model';

@Component({
  selector: 'app-scan-result-skin',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './scan-result.component.html',
  styleUrls: ['./scan-result.component.scss']
})
export class ScanResultComponent implements OnInit, OnDestroy {
  scanId = '';
  scan: ScanDetail | null = null;
  loading = true;
  error = '';
  private pollSubscription?: Subscription;

  constructor(private route: ActivatedRoute, private router: Router, private scanService: ScanService) {}

  ngOnInit() {
    this.route.params.subscribe(params => {
      this.scanId = params['id'];
      if (this.scanId) { this.loadScan(); }
    });
  }

  ngOnDestroy() { this.pollSubscription?.unsubscribe(); }

  loadScan() {
    this.loading = true;
    this.scanService.getScanById(this.scanId).subscribe({
      next: (scan) => {
        this.scan = scan;
        this.loading = false;
        if (scan.status === 'running' || scan.status === 'pending') { this.startPolling(); }
      },
      error: () => { this.error = 'Failed to load scan results'; this.loading = false; }
    });
  }

  startPolling() {
    this.pollSubscription?.unsubscribe();
    this.pollSubscription = this.scanService.pollScanStatus(this.scanId).subscribe({
      next: (scan) => { this.scan = scan; },
      error: () => {}
    });
  }

  getStatusClass(status: string): string { return `status-${status}`; }
  getProgressPercent(): number { if (!this.scan?.total_hosts || !this.scan?.scanned_hosts) return 0; return (this.scan.scanned_hosts / this.scan.total_hosts) * 100; }
  formatDate(dateString?: string): string { if (!dateString) return '-'; return new Date(dateString).toLocaleString(); }
  getDuration(): string { if (!this.scan?.started_at) return '-'; const start = new Date(this.scan.started_at).getTime(); const end = this.scan.completed_at ? new Date(this.scan.completed_at).getTime() : Date.now(); const seconds = Math.floor((end - start)/1000); const minutes = Math.floor(seconds/60); const hours = Math.floor(minutes/60); if (hours>0) return `${hours}h ${minutes%60}m`; if (minutes>0) return `${minutes}m ${seconds%60}s`; return `${seconds}s`; }
  backToManager() { this.router.navigate(['/scan-manager']); }
}


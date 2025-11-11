import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-settings-skin',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss']
})
export class SettingsComponent implements OnInit {
  apiKey = '';
  savedKey = '';
  showKey = false;
  saveMessage = '';

  mockSettings = { slack_webhook: '', telegram_token: '', ai_sensitivity: 5, threat_intel_enabled: true, auto_scan_enabled: false };

  ngOnInit() {
    const storedKey = localStorage.getItem('cortex_api_key');
    if (storedKey) { this.savedKey = storedKey; this.apiKey = storedKey; }
  }

  saveApiKey() {
    if (!this.apiKey.trim()) { this.saveMessage = 'Please enter a valid API key'; setTimeout(() => this.saveMessage = '', 3000); return; }
    localStorage.setItem('cortex_api_key', this.apiKey);
    this.savedKey = this.apiKey;
    this.saveMessage = 'API key saved successfully';
    setTimeout(() => this.saveMessage = '', 3000);
  }

  toggleKeyVisibility() { this.showKey = !this.showKey; }
  clearApiKey() { localStorage.removeItem('cortex_api_key'); this.apiKey = ''; this.savedKey = ''; this.saveMessage = 'API key cleared'; setTimeout(() => this.saveMessage = '', 3000); }

  get maskedKey(): string { if (!this.savedKey) return ''; return '•'.repeat(this.savedKey.length); }
  get displayKey(): string { return this.showKey ? this.savedKey : this.maskedKey; }
}


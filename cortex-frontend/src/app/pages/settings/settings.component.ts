import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { NgIf } from '@angular/common';
import { CardComponent } from '../../shared/card/card.component';

@Component({
  selector: 'app-settings',
  standalone: true,
  imports: [FormsModule, NgIf, CardComponent],
  templateUrl: './settings.component.html',
  styleUrl: './settings.component.scss'
})
export class SettingsComponent {
  key = localStorage.getItem('cortex_api_key') || '';
  saved = false;

  save() {
    localStorage.setItem('cortex_api_key', this.key.trim());
    this.saved = true;
    setTimeout(() => (this.saved = false), 1500);
  }
}

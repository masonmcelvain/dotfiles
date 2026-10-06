import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';
import * as Main from 'resource:///org/gnome/shell/ui/main.js';

export default class HideDateMenuCards extends Extension {
    enable() {
        const {dateMenu} = Main.panel.statusArea;
        this._items = [dateMenu._eventsItem, dateMenu._clocksItem];

        // Each card's _sync() re-shows it when calendars or installed apps
        // change, so hide it again whenever that happens.
        this._visibleIds = this._items.map(item => {
            item.hide();
            return item.connect('notify::visible', () => {
                if (item.visible)
                    item.hide();
            });
        });
    }

    disable() {
        this._items.forEach((item, i) => {
            item.disconnect(this._visibleIds[i]);
            // Let the card decide its own visibility again.
            item._sync();
        });
        this._items = null;
        this._visibleIds = null;
    }
}

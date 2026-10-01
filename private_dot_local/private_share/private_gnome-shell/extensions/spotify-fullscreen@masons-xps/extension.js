import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';

export default class SpotifyFullscreen extends Extension {
    enable() {
        this._windowCreatedId = global.display.connect('window-created', (_display, window) => {
            const actor = window.get_compositor_private();
            if (!actor)
                return;

            // Wait for the first frame so Spotify has finished setting up its
            // window (including its saved size) before we fullscreen it.
            const frameId = actor.connect('first-frame', () => {
                actor.disconnect(frameId);
                if (window.get_wm_class()?.toLowerCase() === 'spotify')
                    window.make_fullscreen();
            });
        });
    }

    disable() {
        global.display.disconnect(this._windowCreatedId);
        this._windowCreatedId = null;
    }
}

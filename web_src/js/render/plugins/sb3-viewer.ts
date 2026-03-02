import type {FileRenderPlugin} from '../plugin.ts';

export function newRenderPluginSb3Viewer(): FileRenderPlugin {
  return {
    name: 'sb3-viewer',

    canHandle(filename: string, _mimeType: string): boolean {
      return filename.toLowerCase().endsWith('.sb3');
    },

    async render(container: HTMLElement, fileUrl: string, options?: {username?: string}): Promise<void> {
      // Clear container content
      container.innerHTML = '';
      
      // Build absolute URL if fileUrl is relative
      const absoluteUrl = fileUrl.startsWith('http') ? fileUrl : `${window.location.origin}${fileUrl}`;
      
      // Remove protocol from URL for Turbowarp (keep only domain + path)
      // Format: domain.com/path/to/file.sb3 (no http:// or https://)
      const urlWithoutProtocol = absoluteUrl.replace(/^https?:\/\//, '');
      
      // Determine username for TurboWarp
      let twUsername = options?.username || '';
      if (!twUsername || twUsername === 'visitor') {
        twUsername = `visitor${Math.floor(Math.random() * 1000000)}`;
      }
      
      // Create iframe for TurboWarp embed
      const iframe = document.createElement('iframe');
      iframe.src = `https://turbowarp.org/embed?project_url=${encodeURIComponent(urlWithoutProtocol)}&username=${encodeURIComponent(twUsername)}`;
      iframe.width = '482';
      iframe.height = '412';
      iframe.setAttribute('allowtransparency', 'true');
      iframe.setAttribute('frameborder', '0');
      iframe.setAttribute('scrolling', 'no');
      iframe.setAttribute('allowfullscreen', '');
      iframe.style.colorScheme = 'auto';
      iframe.style.display = 'block';
      iframe.style.margin = '0 auto';
      iframe.style.border = 'none';
      
      // Add iframe to container
      container.appendChild(iframe);
    },
  };
}
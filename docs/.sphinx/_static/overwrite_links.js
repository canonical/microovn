 // Replaces oldDomain with newDomain in relevant anchor tags
 const RtDHostedDomain = 'canonical-microovn-documentation.readthedocs-hosted.com';
 const newDomain = 'ubuntu.com/docs/microovn';

 function escapeRegExp(value) {
     return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
 }

 function overwriteMatchingAnchorUrls(container) {
     if (!container) return;

     const anchors = container.querySelectorAll('a[href], link[href]');
     const RtDHostedDomainRegex = new RegExp(escapeRegExp(RtDHostedDomain), 'g');

     anchors.forEach(anchor => {
         anchor.href = anchor.href.replace(RtDHostedDomainRegex, newDomain);
     });
 }

 overwriteMatchingAnchorUrls(document.querySelector('header'));

 // Use a MutationObserver to wait for the RTD flyout element to appear in the DOM
 const observer = new MutationObserver(function(mutations, obs) {

     const rtdFlyout = document.querySelector('readthedocs-flyout');
     if (!rtdFlyout) return;

     obs.disconnect();

     rtdFlyout.addEventListener('click', function() {
         const shadowRoot = rtdFlyout.shadowRoot;
         if (!shadowRoot) return;

         overwriteMatchingAnchorUrls(shadowRoot);
     });
 });

 observer.observe(document.body, { childList: true, subtree: true });

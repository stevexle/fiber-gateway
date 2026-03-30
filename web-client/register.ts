import axios from 'axios';

const API_BASE = 'http://localhost:8080/api/v1';

const form = document.getElementById('register-form') as HTMLFormElement;
const resultCard = document.getElementById('result') as HTMLElement;
const clientData = document.getElementById('client-data') as HTMLElement;

form.addEventListener('submit', async (e) => {
    e.preventDefault();

    const name = (document.getElementById('name') as HTMLInputElement).value;
    const realmId = (document.getElementById('realm_id') as HTMLInputElement).value;
    const description = (document.getElementById('description') as HTMLTextAreaElement).value;
    const iconUrl = (document.getElementById('icon_url') as HTMLInputElement).value;
    const homePageUrl = (document.getElementById('home_page_url') as HTMLInputElement).value;
    const privacyPolicyUrl = (document.getElementById('privacy_policy_url') as HTMLInputElement).value;
    const redirectUris = (document.getElementById('redirect_uris') as HTMLInputElement).value;
    const logoutUris = (document.getElementById('logout_uris') as HTMLInputElement).value;
    const webOrigins = (document.getElementById('web_origins') as HTMLInputElement).value;
    const isConfidential = (document.getElementById('is_confidential') as HTMLInputElement).checked;

    // Slug-style Client ID
    const clientId = name.toLowerCase().replace(/ /g, '-').replace(/[^\w-]+/g, '') + '-' + Math.floor(Math.random() * 1000);

    try {
        const res = await axios.post(`${API_BASE}/client/register`, {
            client_id: clientId,
            name: name,
            realm_id: realmId,
            description: description,
            is_confidential: isConfidential,
            icon_url: iconUrl,
            home_page_url: homePageUrl,
            privacy_policy_url: privacyPolicyUrl,
            sign_in_redirect_uris: redirectUris,
            sign_out_redirect_uris: logoutUris,
            web_origins: webOrigins
        });

        // Show result
        resultCard.style.display = 'block';
        clientData.innerText = JSON.stringify(res.data, null, 2);
        resultCard.scrollIntoView({ behavior: 'smooth' });

    } catch (err: any) {
        alert('Provisioning failed: ' + (err.response?.data?.error || err.message));
    }
});

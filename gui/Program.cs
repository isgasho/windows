using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using System.Windows.Forms;

namespace NextDNS
{
    static class Program
    {
        private static SettingsForm settings;

        /// <summary>
        /// The main entry point for the application.
        /// </summary>
        [STAThread]
        static void Main()
        {
            using (Mutex mutex = new Mutex(false, "Global\\NextDNS"))
            {
                if (!mutex.WaitOne(0, false))
                {
                    // Another instance of the app is already running. Instead of running, send
                    // a message thru the service so the main app opens it's windows.
                    var service = new Service.Client();
                    service.SendAsync(new Service.Event("open")).GetAwaiter().GetResult();
                    return;
                }

                Application.EnableVisualStyles();
                Application.SetCompatibleTextRenderingDefault(false);
                settings = new SettingsForm();
                Application.Run();
            }
        }
    }
}

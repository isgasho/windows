using System;
using System.IO;
using System.IO.Pipes;
using System.Text;
using System.Diagnostics;
using System.Runtime.Serialization;
using System.Runtime.Serialization.Json;
using System.Threading.Tasks;
using System.Threading;

namespace NextDNS
{
    namespace Service
    {
        [DataContract]
        class Event
        {
            [DataMember(IsRequired=true)]
            public string name;

            [DataMember]
            public EventData data;

            public Event(string name)
            {
                this.name = name;
            }
        }
        [DataContract]
        class EventData
        {
            [DataMember]
            public bool enabled;

            [DataMember]
            public string state;

            [DataMember]
            public string error;

            [DataMember]
            public string configuration;

            [DataMember]
            public bool reportDeviceName;

            [DataMember]
            public bool checkUpdates;

            [DataMember]
            public string updateChannel;
        }
        class Client
        {
            private Thread thread;
            public event EventHandler<Event> EventReceived;
            public event EventHandler Connected;

            public void Connect()
            {
                thread = new Thread(new ThreadStart(Run));
                thread.Start();
            }

            private void Run()
            {
                while (true)
                {
                    Debug.WriteLine("Connecting to service");
                    using (var pipe = new NamedPipeClientStream(".", "NextDNS", PipeDirection.In, PipeOptions.None))
                    {
                        try
                        {
                            pipe.Connect(5000);
                            Connected?.Invoke(this, EventArgs.Empty);
                            ReadLoop(pipe);
                        }
                        catch (Exception e)
                        {
                            Debug.WriteLine("NamedPipe connect: {0}", (object)e.Message);
                            Thread.Sleep(500);
                        }
                    }
                }
            }

            private void ReadLoop(NamedPipeClientStream pipe)
            {
                var ser = new DataContractJsonSerializer(typeof(Event));
                using (var r = new StreamReader(pipe))
                {
                    string line;
                    while ((line = r.ReadLine()) != null)
                    {
                        using (var s = new MemoryStream(Encoding.UTF8.GetBytes(line)))
                        {
                            var e = (Event)ser.ReadObject(s);
                            Debug.WriteLine("Event received: {0}", (object)e.name);
                            EventReceived?.Invoke(this, e);
                        }
                    }
                    Debug.WriteLine("Pipe was closed");
                }
            }

            async public Task SendAsync(Event e)
            {
                Debug.WriteLine("Event sent: {0}", (object)e.name);
                using (var pipe = new NamedPipeClientStream(".", "NextDNS", PipeDirection.Out, PipeOptions.None))
                {
                    await pipe.ConnectAsync(1000).ConfigureAwait(false);
                    var ms = new MemoryStream();
                    var ser = new DataContractJsonSerializer(typeof(Event));
                    ser.WriteObject(ms, e);
                    ms.WriteByte(Convert.ToByte('\n'));
                    byte[] json = ms.ToArray();
                    ms.Close();
                    await pipe.WriteAsync(json, 0, json.Length).ConfigureAwait(false);
                }
            }
        }
    }
}

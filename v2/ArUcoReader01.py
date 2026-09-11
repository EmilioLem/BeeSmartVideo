import cv2
import numpy as np
import time
import csv
import os
from collections import deque
from picamera2 import Picamera2

# --- FUNCIÓN DE NOMENCLATURA (H1 a H5) ---
def obtener_nomenclatura(id_entero):
    # Divide en bloques de 2000 para obtener la colmena (H1, H2, H3, H4, H5)
    grupo = (id_entero // 2000) + 1
    # Obtiene el número de individuo dentro de esa colmena (0 a 1999)
    individuo = id_entero % 2000
    return f"H{grupo}-{individuo:04d}"

# 1. RUTAS
DIRECTORIO_BASE = os.path.dirname(os.path.abspath(__file__))
archivo_crudo = os.path.join(DIRECTORIO_BASE, "registro_basico.csv")
archivo_detallado = os.path.join(DIRECTORIO_BASE, "registro_completo.csv")

for arc, head in [(archivo_crudo, ["Timestamp", "ID_Abeja", "Evento"]), 
                  (archivo_detallado, ["Timestamp", "ID_Abeja", "Evento", "T_Transito_Seg", "T_Afuera_Seg", "T_Adentro_Seg"])]:
    if not os.path.exists(arc):
        with open(arc, 'w', newline='') as f:
            csv.writer(f).writerow(head)

# 2. CONFIGURACIÓN DE CÁMARA (1080p, Filtro Rojo)
picam2 = Picamera2()
config = picam2.create_preview_configuration(main={"size": (1920, 1080), "format": "BGR888"})
picam2.configure(config)
picam2.start()

picam2.set_controls({
    "FrameRate": 30, 
    "Sharpness": 1.5, 
    "Contrast": 1.5,
    "AfMode": 0,          
    "LensPosition": 6.0,  
    "AwbMode": 0,         
    "ColourGains": (2.0, 1.0) 
})

# 3. PARÁMETROS DE ARUCO (10,000 Marcadores)
aruco_dict = cv2.aruco.extendDictionary(10000, 5) 
aruco_params = cv2.aruco.DetectorParameters()
aruco_params.minMarkerPerimeterRate = 0.010 
aruco_params.cornerRefinementMethod = cv2.aruco.CORNER_REFINE_SUBPIX 
detector = cv2.aruco.ArucoDetector(aruco_dict, aruco_params)

trayectorias = {}; tiempos_visto = {}; ultimo_visto = {}; memoria_colmena = {}
MAX_LONGITUD_TRAYECTORIA = 300 
LIMITE_INACTIVIDAD = 2.0      
DESPLAZAMIENTO_MINIMO = 200  
LINEA_COLMENA = 20           
LINEA_CAMPO = 1900           

# 4. CICLO PRINCIPAL (Modo Demonio)
print("SISTEMA INICIADO: Demonio Bee Smart")
print(f"Directorio de guardado: {DIRECTORIO_BASE}")

try:
    while True:
        frame_bgr = picam2.capture_array()
        
        # Extraemos solo el canal rojo para máxima nitidez bajo el LED rojo
        canal_rojo = frame_bgr[:, :, 2]
        t_actual = time.time()
        
        corners, ids, rejected = detector.detectMarkers(canal_rojo)

        if ids is not None:
            for i, id_arr in enumerate(ids):
                id_b = int(id_arr[0]) 
                cX = int(corners[i][0][:,0].mean())
                
                if id_b not in tiempos_visto:
                    tiempos_visto[id_b] = t_actual
                    trayectorias[id_b] = deque(maxlen=MAX_LONGITUD_TRAYECTORIA)
                
                trayectorias[id_b].append(cX)
                ultimo_visto[id_b] = t_actual

        ids_a_borrar = []
        for id_b in list(ultimo_visto.keys()):
            if t_actual - ultimo_visto[id_b] > LIMITE_INACTIVIDAD:
                pts = list(trayectorias[id_b])
                dur_cam = ultimo_visto[id_b] - tiempos_visto[id_b]
                
                if len(pts) >= 2 and dur_cam >= 0.1:
                    primer_x = pts[0]
                    ultimo_x = pts[-1]
                    delta_x = ultimo_x - primer_x
                    distancia = abs(delta_x)
                    
                    if distancia >= DESPLAZAMIENTO_MINIMO:
                        ts = time.strftime("%Y-%m-%d %H:%M:%S")
                        ev = None
                        
                        if delta_x < 0 and ultimo_x <= LINEA_COLMENA:
                            ev = "ENTRO_COLMENA"
                        elif delta_x > 0 and ultimo_x >= LINEA_CAMPO:
                            ev = "SALIO_AL_CAMPO"

                        if ev:
                            t_af, t_ad = "N/A", "N/A"
                            msg_extra = " (Primer registro)"
                            
                            if ev == "ENTRO_COLMENA":
                                if id_b in memoria_colmena and memoria_colmena[id_b]["estado"] == "AFUERA":
                                    t_af = round(t_actual - memoria_colmena[id_b]["ts_evento"], 2)
                                    msg_extra = f" | Afuera: {t_af}s"
                                memoria_colmena[id_b] = {"estado": "ADENTRO", "ts_evento": t_actual}
                            else:
                                if id_b in memoria_colmena and memoria_colmena[id_b]["estado"] == "ADENTRO":
                                    t_ad = round(t_actual - memoria_colmena[id_b]["ts_evento"], 2)
                                    msg_extra = f" | Adentro: {t_ad}s"
                                memoria_colmena[id_b] = {"estado": "AFUERA", "ts_evento": t_actual}

                            # --- CONVERSIÓN A NOMENCLATURA PARA LA BASE DE DATOS ---
                            id_etiqueta = obtener_nomenclatura(id_b)

                            # Escritura en CSVs
                            with open(archivo_detallado, 'a', newline='') as f:
                                csv.writer(f).writerow([ts, id_etiqueta, ev, round(dur_cam, 2), t_af, t_ad])
                                
                            with open(archivo_crudo, 'a', newline='') as f:
                                csv.writer(f).writerow([ts, id_etiqueta, ev])
                            
                            print(f"✅ ID:{id_etiqueta} | {ev}{msg_extra} | Tránsito: {round(dur_cam, 2)}s")

                ids_a_borrar.append(id_b)

        for id_b in ids_a_borrar:
            for d in [tiempos_visto, ultimo_visto, trayectorias]: d.pop(id_b, None)

        # Pequeña pausa para no saturar la CPU al 100%
        time.sleep(0.001)

except KeyboardInterrupt:
    print("\nApagando demonio...")
except Exception as e:
    print(f"\nError inesperado: {e}")
finally:
    picam2.stop()
    print("Conexión de cámara cerrada.")
